package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"fixphone/backend/internal/config"
	"fixphone/backend/internal/database"
	"fixphone/backend/internal/pkg/version"

	gormlogger "gorm.io/gorm/logger"
)

var roleNamePattern = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "migrate":
		err = runMigrate(os.Args[2:])
	case "seed":
		err = runSeed(os.Args[2:])
	case "create-tenant":
		err = runCreateTenant(os.Args[2:])
	case "db-create":
		err = runDBCreate(os.Args[2:])
	case "db-drop":
		err = runDBDrop(os.Args[2:])
	case "db-setup":
		err = runDBSetup(os.Args[2:])
	case "set-app-role":
		err = runSetAppRole(os.Args[2:])
	case "doctor":
		err = runDoctor(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Printf("fixphone-cli %s (%s)\n", version.Version, version.Commit)
	case "help", "--help", "-h":
		usage()
	default:
		err = fmt.Errorf("perintah tidak dikenal: %s", os.Args[1])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`FixPhone CLI

Usage:
  fixphone-cli migrate up [--steps N]
  fixphone-cli migrate down [--steps N]
  fixphone-cli migrate status
  fixphone-cli migrate force --version N
  fixphone-cli seed
  fixphone-cli create-tenant --name NAME --slug SLUG --owner-email EMAIL --owner-password PASSWORD [flags]
  fixphone-cli db-create [--name DATABASE]
  fixphone-cli db-drop --name DATABASE --yes [--force]
  fixphone-cli db-setup
  fixphone-cli set-app-role [--role fixphone_app]
  fixphone-cli doctor
  fixphone-cli version

Environment:
  DB_DSN is required. DB_ADMIN_DSN (schema owner) is required for migration,
  seeding, set-app-role, and doctor in production. SEED_SUPERADMIN_EMAIL and
  SEED_SUPERADMIN_PASSWORD are required for "seed". APP_DB_PASSWORD is required
  for "set-app-role". Configuration is loaded from .env automatically.
`)
}

func runMigrate(args []string) error {
	if len(args) < 1 {
		return errors.New("gunakan: migrate up|down|status|force")
	}
	action := args[0]
	fs := flag.NewFlagSet("migrate "+action, flag.ContinueOnError)
	steps := fs.Int("steps", 0, "jumlah migrasi (0 = semua untuk down)")
	forceVersion := fs.Int("version", -1, "versi yang dipaksa untuk force")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	cfg, err := config.LoadDBOnly()
	if err != nil {
		return err
	}
	migrator, err := database.NewMigrator(cfg.MigrationDSN())
	if err != nil {
		return err
	}
	defer migrator.Close()

	switch action {
	case "up":
		if err := migrator.UpSteps(*steps); err != nil {
			return fmt.Errorf("migrate up: %w", err)
		}
		v, dirty, err := migrator.Status()
		if err != nil {
			return err
		}
		fmt.Printf("migrate up selesai. versi=%d dirty=%t expected=%d\n", v, dirty, database.ExpectedVersion)
	case "down":
		if *steps < 0 {
			return errors.New("--steps tidak boleh negatif")
		}
		if err := migrator.Down(*steps); err != nil {
			return fmt.Errorf("migrate down: %w", err)
		}
		v, dirty, err := migrator.Status()
		if err != nil {
			return err
		}
		fmt.Printf("migrate down selesai. versi=%d dirty=%t\n", v, dirty)
	case "status":
		v, dirty, err := migrator.Status()
		if err != nil {
			return err
		}
		fmt.Printf("versi=%d dirty=%t expected=%d\n", v, dirty, database.ExpectedVersion)
		if dirty {
			fmt.Println("PERINGATAN: migrasi dirty. Perbaiki lalu jalankan 'migrate force --version N'.")
		}
		if v < database.ExpectedVersion && !dirty {
			fmt.Println("Database belum berada pada versi terbaru. Jalankan 'migrate up'.")
		}
	case "force":
		if *forceVersion < 0 {
			return errors.New("--version wajib diisi untuk force")
		}
		if err := migrator.Force(*forceVersion); err != nil {
			return fmt.Errorf("migrate force: %w", err)
		}
		fmt.Printf("migrate force selesai. versi=%d\n", *forceVersion)
	default:
		return fmt.Errorf("aksi migrate tidak dikenal: %s", action)
	}
	return nil
}

func runSeed(args []string) error {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := config.LoadDBOnly()
	if err != nil {
		return err
	}

	email := strings.TrimSpace(os.Getenv("SEED_SUPERADMIN_EMAIL"))
	password := os.Getenv("SEED_SUPERADMIN_PASSWORD")
	name := strings.TrimSpace(os.Getenv("SEED_SUPERADMIN_NAME"))
	if email == "" || password == "" {
		return errors.New("SEED_SUPERADMIN_EMAIL dan SEED_SUPERADMIN_PASSWORD wajib diisi di environment")
	}

	db, err := database.Open(context.Background(), cfg.MigrationDSN(), database.DefaultOpenOptions())
	if err != nil {
		return err
	}
	defer database.Close(db)

	result, err := database.Seed(context.Background(), db, database.SeedOptions{
		SuperadminName:     name,
		SuperadminEmail:    email,
		SuperadminPassword: password,
	})
	if err != nil {
		return err
	}
	fmt.Printf("seed selesai (idempoten). roles=%d permissions=%d role_permissions=%d superadmins=%d expense_categories=%d\n",
		result.Roles, result.Permissions, result.RolePermissions, result.Superadmins, result.ExpenseCategories)
	return nil
}

func runCreateTenant(args []string) error {
	fs := flag.NewFlagSet("create-tenant", flag.ContinueOnError)
	name := fs.String("name", "", "nama badan usaha")
	slug := fs.String("slug", "", "slug unik tenant")
	ownerName := fs.String("owner-name", "Owner", "nama owner")
	ownerEmail := fs.String("owner-email", "", "email login owner")
	ownerPassword := fs.String("owner-password", "", "password owner (minimal 8 karakter, huruf + angka)")
	outletName := fs.String("outlet-name", "Outlet Utama", "nama outlet utama")
	outletCode := fs.String("outlet-code", "MAIN", "kode outlet utama")
	timezone := fs.String("timezone", "Asia/Jakarta", "zona waktu outlet")
	phone := fs.String("phone", "", "telepon tenant/outlet")
	address := fs.String("address", "", "alamat outlet")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.LoadDBOnly()
	if err != nil {
		return err
	}
	db, err := database.Open(context.Background(), cfg.MigrationDSN(), database.DefaultOpenOptions())
	if err != nil {
		return err
	}
	defer database.Close(db)

	result, err := database.CreateTenant(context.Background(), db, database.CreateTenantInput{
		Name:          *name,
		Slug:          *slug,
		OwnerName:     *ownerName,
		OwnerEmail:    *ownerEmail,
		OwnerPassword: *ownerPassword,
		OutletName:    *outletName,
		OutletCode:    *outletCode,
		Timezone:      *timezone,
		Phone:         *phone,
		Address:       *address,
	})
	if err != nil {
		return err
	}
	fmt.Printf("tenant dibuat. tenant_id=%s outlet_id=%s owner_id=%s owner_email=%s\n",
		result.TenantID, result.OutletID, result.OwnerID, strings.ToLower(*ownerEmail))
	return nil
}

func runDBCreate(args []string) error {
	fs := flag.NewFlagSet("db-create", flag.ContinueOnError)
	name := fs.String("name", "", "nama database (default: dari DB_DSN)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := config.LoadDBOnly()
	if err != nil {
		return err
	}
	if *name == "" {
		*name, err = database.ParseDatabaseName(cfg.DBDSN)
		if err != nil {
			return err
		}
	}
	created, err := database.EnsureDatabase(context.Background(), cfg.MigrationDSN(), *name)
	if err != nil {
		return err
	}
	if created {
		fmt.Printf("database %s dibuat\n", *name)
	} else {
		fmt.Printf("database %s sudah ada\n", *name)
	}
	return nil
}

func runDBDrop(args []string) error {
	fs := flag.NewFlagSet("db-drop", flag.ContinueOnError)
	name := fs.String("name", "", "nama database")
	yes := fs.Bool("yes", false, "konfirmasi penghapusan")
	force := fs.Bool("force", false, "izinkan nama selain _test/_dev")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" {
		return errors.New("--name wajib diisi")
	}
	if !*yes {
		return errors.New("tambahkan --yes untuk mengonfirmasi penghapusan database")
	}
	cfg, err := config.LoadDBOnly()
	if err != nil {
		return err
	}
	if err := database.DropDatabase(context.Background(), cfg.MigrationDSN(), *name, *force); err != nil {
		return err
	}
	fmt.Printf("database %s dihapus\n", *name)
	return nil
}

func runDBSetup(args []string) error {
	fs := flag.NewFlagSet("db-setup", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := config.LoadDBOnly()
	if err != nil {
		return err
	}
	name, err := database.ParseDatabaseName(cfg.DBDSN)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	created, err := database.EnsureDatabase(ctx, cfg.MigrationDSN(), name)
	if err != nil {
		return err
	}
	fmt.Printf("database utama %s: %s\n", name, createdLabel(created))

	testName := testDatabaseName(name)
	created, err = database.EnsureDatabase(ctx, cfg.MigrationDSN(), testName)
	if err != nil {
		return err
	}
	fmt.Printf("database test %s: %s\n", testName, createdLabel(created))

	testDSN, err := database.DSNWithDatabase(cfg.DBDSN, testName)
	if err != nil {
		return err
	}
	fmt.Printf("TEST_DB_DSN=%s\n", testDSN)
	return nil
}

// runSetAppRole enables LOGIN for the RLS application role and sets its
// password. It must run with the schema owner (DB_ADMIN_DSN) because the role
// is created NOLOGIN by the migration. The password is passed to PostgreSQL
// with format(%L), so it is safely quoted and never written to logs.
func runSetAppRole(args []string) error {
	fs := flag.NewFlagSet("set-app-role", flag.ContinueOnError)
	role := fs.String("role", database.AppRoleName, "nama role aplikasi")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !roleNamePattern.MatchString(*role) {
		return errors.New("nama role hanya boleh huruf kecil/angka/underscore")
	}
	password := os.Getenv("APP_DB_PASSWORD")
	if len(password) < 16 {
		return errors.New("APP_DB_PASSWORD wajib diisi minimal 16 karakter")
	}
	if len(password) > 1024 {
		return errors.New("APP_DB_PASSWORD terlalu panjang (maksimal 1024 karakter)")
	}
	if strings.TrimSpace(os.Getenv("DB_ADMIN_DSN")) == "" {
		return errors.New("set-app-role memerlukan DB_ADMIN_DSN (pemilik skema), bukan DB_DSN aplikasi")
	}

	cfg, err := config.LoadDBOnly()
	if err != nil {
		return err
	}
	db, err := sql.Open("pgx", cfg.MigrationDSN())
	if err != nil {
		return fmt.Errorf("membuka koneksi admin: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var exists bool
	if err := db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, *role).Scan(&exists); err != nil {
		return fmt.Errorf("memeriksa role: %w", err)
	}
	if !exists {
		return fmt.Errorf("role %s belum ada; jalankan 'migrate up' terlebih dahulu", *role)
	}

	var stmt string
	if err := db.QueryRowContext(ctx,
		`SELECT format('ALTER ROLE %I WITH LOGIN PASSWORD %L', $1, $2)`,
		*role, password).Scan(&stmt); err != nil {
		return fmt.Errorf("menyiapkan ALTER ROLE: %w", err)
	}
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("mengaktifkan role %s: %w", *role, err)
	}
	fmt.Printf("role %s: LOGIN aktif, password diperbarui\n", *role)
	return nil
}

// runDoctor inspects a database for the checks that commonly break a
// production deployment: migration state, tenant RLS policies, application
// role attributes, and leftover demo data.
func runDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	role := fs.String("role", database.AppRoleName, "nama role aplikasi")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !roleNamePattern.MatchString(*role) {
		return errors.New("nama role hanya boleh huruf kecil/angka/underscore")
	}

	cfg, err := config.LoadDBOnly()
	if err != nil {
		return err
	}
	db, err := sql.Open("pgx", cfg.MigrationDSN())
	if err != nil {
		return fmt.Errorf("membuka koneksi database: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	problems := make([]string, 0)
	warnings := make([]string, 0)

	version, dirty, err := database.ReadMigrationState(ctx, db)
	if err != nil {
		return fmt.Errorf("membaca status migrasi: %w", err)
	}
	switch {
	case version == 0:
		problems = append(problems, "skema belum dimigrasi; jalankan 'migrate up'")
	case version < database.ExpectedVersion:
		problems = append(problems, fmt.Sprintf("versi migrasi %d lebih lama dari %d; jalankan 'migrate up'", version, database.ExpectedVersion))
	case version > database.ExpectedVersion:
		warnings = append(warnings, fmt.Sprintf("versi migrasi %d lebih baru dari versi binary %d", version, database.ExpectedVersion))
	}
	if dirty {
		problems = append(problems, "migrasi berstatus dirty; perbaiki lalu jalankan 'migrate force --version N'")
	}

	var rlsTables, policies int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*) FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND c.relkind = 'r' AND c.relrowsecurity`).Scan(&rlsTables); err != nil {
		return fmt.Errorf("menghitung tabel RLS: %w", err)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM pg_policies WHERE schemaname = 'public'`).Scan(&policies); err != nil {
		return fmt.Errorf("menghitung policy RLS: %w", err)
	}
	if rlsTables < database.ExpectedTenantPolicies || policies < database.ExpectedTenantPolicies {
		problems = append(problems, fmt.Sprintf(
			"RLS belum lengkap: %d tabel aktif, %d policy (diharapkan minimal %d)",
			rlsTables, policies, database.ExpectedTenantPolicies))
	}

	var canLogin, isSuper, bypassRLS bool
	err = db.QueryRowContext(ctx,
		`SELECT rolcanlogin, rolsuper, rolbypassrls FROM pg_roles WHERE rolname = $1`, *role).
		Scan(&canLogin, &isSuper, &bypassRLS)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		problems = append(problems, fmt.Sprintf("role %s belum ada; jalankan 'migrate up'", *role))
	case err != nil:
		return fmt.Errorf("memeriksa role: %w", err)
	default:
		if !canLogin {
			problems = append(problems, fmt.Sprintf("role %s NOLOGIN; jalankan 'set-app-role'", *role))
		}
		if isSuper || bypassRLS {
			problems = append(problems, fmt.Sprintf("role %s dapat melewati RLS (superuser=%t bypassrls=%t)", *role, isSuper, bypassRLS))
		}
	}

	var demoTenants int
	err = db.QueryRowContext(ctx, `
		SELECT count(*) FROM tenants WHERE slug ILIKE '%demo%' OR name ILIKE '%demo%'`).Scan(&demoTenants)
	switch {
	case err != nil && !strings.Contains(strings.ToLower(err.Error()), "does not exist"):
		return fmt.Errorf("memeriksa data demo: %w", err)
	case err == nil && demoTenants > 0:
		warnings = append(warnings, fmt.Sprintf("%d tenant tampak seperti data demo; pastikan tidak dipakai di production", demoTenants))
	}

	dsnUser := "-"
	if u, parseErr := urlUser(cfg.DBDSN); parseErr == nil {
		dsnUser = u
	}

	fmt.Println("FixPhone doctor")
	fmt.Printf("  database      : %s\n", dsnUser)
	fmt.Printf("  migrasi       : versi=%d expected=%d dirty=%t\n", version, database.ExpectedVersion, dirty)
	fmt.Printf("  RLS           : %d tabel, %d policy\n", rlsTables, policies)
	fmt.Printf("  role aplikasi : %s login=%t super=%t bypassrls=%t\n", *role, canLogin, isSuper, bypassRLS)
	fmt.Printf("  data demo     : %d tenant\n", demoTenants)
	for _, warning := range warnings {
		fmt.Printf("  PERINGATAN: %s\n", warning)
	}
	if len(problems) > 0 {
		for _, problem := range problems {
			fmt.Printf("  MASALAH: %s\n", problem)
		}
		return fmt.Errorf("doctor menemukan %d masalah", len(problems))
	}
	fmt.Println("  hasil         : OK")
	return nil
}

func urlUser(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	if u.User == nil || u.User.Username() == "" {
		return "-", nil
	}
	return u.User.Username(), nil
}

func testDatabaseName(name string) string {
	if strings.HasSuffix(name, "_dev") {
		name = strings.TrimSuffix(name, "_dev")
	}
	return name + "_test"
}

func createdLabel(created bool) string {
	if created {
		return "dibuat"
	}
	return "sudah ada"
}

// Keep the GORM logger import explicit so the CLI never logs SQL at info level.
var _ = gormlogger.Silent
