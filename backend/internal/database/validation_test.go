package database_test

import (
	"strings"
	"testing"

	"fixphone/backend/internal/database"
)

func TestNormalizeEmail(t *testing.T) {
	valid := map[string]string{
		"  Owner@Example.COM ":  "owner@example.com",
		"a.b+tag@example.co.id": "a.b+tag@example.co.id",
	}
	for input, want := range valid {
		got, err := database.NormalizeEmail(input)
		if err != nil {
			t.Fatalf("NormalizeEmail(%q): %v", input, err)
		}
		if got != want {
			t.Fatalf("NormalizeEmail(%q) = %q, want %q", input, got, want)
		}
	}

	invalid := []string{"", "not-an-email", "a@", "@example.com", "a b@example.com", strings.Repeat("a", 150) + "@example.com"}
	for _, input := range invalid {
		if _, err := database.NormalizeEmail(input); err == nil {
			t.Fatalf("NormalizeEmail(%q) should fail", input)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	valid := []string{"Rahasia123", "abc12345", "P4ssword!"}
	for _, input := range valid {
		if err := database.ValidatePassword(input); err != nil {
			t.Fatalf("ValidatePassword(%q): %v", input, err)
		}
	}

	invalid := []string{"short1", "onlyletters", "12345678", strings.Repeat("a1", 40)}
	for _, input := range invalid {
		if err := database.ValidatePassword(input); err == nil {
			t.Fatalf("ValidatePassword(%q) should fail", input)
		}
	}
}

func TestParseDatabaseName(t *testing.T) {
	name, err := database.ParseDatabaseName("postgres://postgres@127.0.0.1:55432/fixphone_dev?sslmode=disable")
	if err != nil || name != "fixphone_dev" {
		t.Fatalf("ParseDatabaseName = %q, %v", name, err)
	}
	if _, err := database.ParseDatabaseName("mysql://root@localhost/app"); err == nil {
		t.Fatal("expected scheme validation error")
	}
	if _, err := database.ParseDatabaseName("postgres://postgres@127.0.0.1:55432/"); err == nil {
		t.Fatal("expected missing database error")
	}
}

func TestDSNWithDatabase(t *testing.T) {
	dsn := "postgres://postgres@127.0.0.1:55432/fixphone_dev?sslmode=disable"
	got, err := database.DSNWithDatabase(dsn, "fixphone_test")
	if err != nil {
		t.Fatalf("DSNWithDatabase: %v", err)
	}
	if !strings.Contains(got, "/fixphone_test?") {
		t.Fatalf("unexpected DSN: %s", got)
	}
}
