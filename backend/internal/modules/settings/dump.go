package settings

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// DumpStats reports the size and coverage of a completed tenant dump.
type DumpStats struct {
	SizeBytes  int64
	TableCount int
	RowCount   int64
}

// backupTable is one table of the tenant dump. Tables without a tenant_id
// column declare their own predicate through filter.
type backupTable struct {
	name   string
	filter string // fmt template receiving the tenant id literal
}

// backupQuery builds the COPY statement for one table. COPY ... TO STDOUT
// cannot bind parameters, so the tenant id (a validated UUID literal) is
// inlined into the statement.
func (t backupTable) backupQuery(tenantID uuid.UUID) string {
	predicate := fmt.Sprintf("tenant_id = '%s'", tenantID)
	if t.filter != "" {
		predicate = fmt.Sprintf(t.filter, tenantID)
	}
	return fmt.Sprintf("COPY (SELECT * FROM %s WHERE %s) TO STDOUT", t.name, predicate)
}

// TenantDump writes a tenant-scoped SQL dump (COPY text format) gzipped.
type TenantDump struct {
	dsn      string
	dir      string
	fileName string
}

// NewTenantDump builds a dump writer.
func NewTenantDump(dsn, dir, fileName string) *TenantDump {
	return &TenantDump{dsn: dsn, dir: dir, fileName: fileName}
}

// Run dumps every tenant table filtered by tenant_id into the gzipped file.
func (d *TenantDump) Run(ctx context.Context, tenantID uuid.UUID, tables []backupTable) (DumpStats, error) {
	conn, err := pgx.Connect(ctx, d.dsn)
	if err != nil {
		return DumpStats{}, fmt.Errorf("koneksi backup gagal: %w", err)
	}
	defer func() { _ = conn.Close(context.Background()) }()

	if err := os.MkdirAll(d.dir, 0o750); err != nil {
		return DumpStats{}, fmt.Errorf("membuat folder backup gagal: %w", err)
	}
	path := filepath.Join(d.dir, d.fileName)
	// #nosec G304 -- path dibentuk dari folder backup terkonfigurasi dan nama file internal, bukan input pengguna.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return DumpStats{}, fmt.Errorf("membuat file backup gagal: %w", err)
	}
	defer func() { _ = file.Close() }()

	gzipWriter := gzip.NewWriter(file)
	defer func() { _ = gzipWriter.Close() }()

	var stats DumpStats
	buf := &strings.Builder{}
	if _, err := fmt.Fprintf(buf, "-- FixPhone tenant backup\n-- tenant_id=%s\n-- created=%s\n\n",
		tenantID, nowStamp()); err != nil {
		return DumpStats{}, err
	}
	if _, err := io.WriteString(gzipWriter, buf.String()); err != nil {
		return DumpStats{}, err
	}

	for _, table := range tables {
		if strings.TrimSpace(table.name) == "" {
			continue
		}
		tag, copyErr := conn.PgConn().CopyTo(ctx, gzipWriter, table.backupQuery(tenantID))
		if copyErr != nil {
			return stats, fmt.Errorf("dump tabel %s gagal: %w", table.name, copyErr)
		}
		stats.RowCount += tag.RowsAffected()
		stats.TableCount++
		if _, err := io.WriteString(gzipWriter, "\n"); err != nil {
			return DumpStats{}, err
		}
	}

	if err := gzipWriter.Close(); err != nil {
		return DumpStats{}, fmt.Errorf("menutup gzip gagal: %w", err)
	}
	if err := file.Sync(); err != nil {
		return DumpStats{}, fmt.Errorf("sinkronisasi file gagal: %w", err)
	}
	info, err := os.Stat(path)
	if err == nil {
		stats.SizeBytes = info.Size()
	}
	return stats, nil
}

func nowStamp() string { return time.Now().UTC().Format(time.RFC3339) }

func ensureDir(dir string) error {
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("folder backup belum dikonfigurasi")
	}
	return os.MkdirAll(dir, 0o750)
}

func filepathJoin(dir, name string) string { return filepath.Join(dir, name) }

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
