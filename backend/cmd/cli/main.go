package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"fixphone/backend/internal/config"
	"fixphone/backend/internal/database"
	"fixphone/backend/internal/pkg/version"

	gormlogger "gorm.io/gorm/logger"
)

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
  fixphone-cli version

Environment:
  DB_DSN is required. SEED_SUPERADMIN_EMAIL and SEED_SUPERADMIN_PASSWORD are
  required for "seed". Configuration is loaded from .env automatically.
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
	migrator, err := database.NewMigrator(cfg.DBDSN)
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

	db, err := database.Open(context.Background(), cfg.DBDSN, database.DefaultOpenOptions())
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
	db, err := database.Open(context.Background(), cfg.DBDSN, database.DefaultOpenOptions())
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
	created, err := database.EnsureDatabase(context.Background(), cfg.DBDSN, *name)
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
	if err := database.DropDatabase(context.Background(), cfg.DBDSN, *name, *force); err != nil {
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

	created, err := database.EnsureDatabase(ctx, cfg.DBDSN, name)
	if err != nil {
		return err
	}
	fmt.Printf("database utama %s: %s\n", name, createdLabel(created))

	testName := testDatabaseName(name)
	created, err = database.EnsureDatabase(ctx, cfg.DBDSN, testName)
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
