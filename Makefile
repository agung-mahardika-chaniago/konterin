# FixPhone — perintah pengembangan harian.
# Di Windows, jalankan `make <target>` dari PowerShell yang memiliki GNU Make,
# atau salin perintah di bawah secara manual.

GO ?= go
PS ?= powershell
CLI := cd backend && $(GO) run ./cmd/cli
DEV_DB := $(PS) -ExecutionPolicy Bypass -File scripts/dev-db.ps1

.DEFAULT_GOAL := help
.PHONY: help setup deps db-start db-stop db-status db-reset db-setup \
	migrate-up migrate-down migrate-status seed tenant api \
	test vet build check fmt compose-up compose-down compose-logs clean

help:
	@echo "FixPhone development commands"
	@echo "  make setup          # start portable DB, create DBs, download Go deps"
	@echo "  make db-start       # start PostgreSQL portable di 127.0.0.1:55432"
	@echo "  make db-stop        # hentikan PostgreSQL portable"
	@echo "  make db-status      # cek status cluster"
	@echo "  make db-reset       # HAPUS data dev lalu inisialisasi ulang"
	@echo "  make db-setup       # buat fixphone_dev + fixphone_test"
	@echo "  make migrate-up     # jalankan semua migrasi"
	@echo "  make migrate-down   # rollback semua migrasi"
	@echo "  make migrate-status # versi migrasi saat ini"
	@echo "  make seed           # seed role/permission/super admin (idempoten)"
	@echo "  make tenant NAME='Toko A' SLUG=toko-a OWNER_EMAIL=a@toko.id OWNER_PASSWORD='Rahasia123'"
	@echo "  make test           # unit + integration test"
	@echo "  make check          # gofmt + vet + build + test"
	@echo "  make compose-up     # PostgreSQL via Docker (bila tersedia)"

setup: db-start db-setup deps
	@echo "Siap. Jalankan: make migrate-up && make seed"

deps:
	cd backend && $(GO) mod download

db-start:
	$(DEV_DB) start

db-stop:
	$(DEV_DB) stop

db-status:
	$(DEV_DB) status

db-reset:
	$(DEV_DB) reset

db-setup:
	$(CLI) db-setup

migrate-up:
	$(CLI) migrate up

migrate-down:
	$(CLI) migrate down

migrate-status:
	$(CLI) migrate status

seed:
	$(CLI) seed

tenant:
	@if [ -z "$(NAME)" ] || [ -z "$(SLUG)" ] || [ -z "$(OWNER_EMAIL)" ] || [ -z "$(OWNER_PASSWORD)" ]; then \
		echo "Wajib: NAME=... SLUG=... OWNER_EMAIL=... OWNER_PASSWORD=..."; exit 1; \
	fi
	$(CLI) create-tenant --name "$(NAME)" --slug "$(SLUG)" \
		--owner-email "$(OWNER_EMAIL)" --owner-password "$(OWNER_PASSWORD)" \
		--outlet-name "$(or $(OUTLET_NAME),Outlet Utama)" --outlet-code "$(or $(OUTLET_CODE),MAIN)"

api:
	cd backend && $(GO) run ./cmd/api

test:
	cd backend && $(GO) test ./... -count=1

vet:
	cd backend && $(GO) vet ./...

build:
	cd backend && $(GO) build ./...

fmt:
	cd backend && gofmt -w ./cmd ./internal ./migrations ./tests

check: fmt vet build test
	cd backend && test -z "$$(gofmt -l ./cmd ./internal ./migrations ./tests)"

compose-up:
	docker compose up -d

compose-down:
	docker compose down

compose-logs:
	docker compose logs -f db

clean:
	cd backend && $(GO) clean -testcache
