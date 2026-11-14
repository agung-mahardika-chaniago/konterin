package database

import (
	"database/sql"
	"errors"
	"sync"

	"gorm.io/gorm"
)

// AppRoleName is the non-superuser role used by the API in production. It owns
// no tables and cannot bypass row level security.
const AppRoleName = "fixphone_app"

// ExpectedTenantPolicies is the number of tenant_isolation policies created by
// migration 000019. Doctor compares the live database against this number.
const ExpectedTenantPolicies = 30

// boundDatabases tracks handles that already have the tenant binding callbacks
// so repeated router construction (tests rebuild engines frequently) does not
// register duplicate callbacks.
var boundDatabases sync.Map

// EnableTenantBinding rewires every statement whose context carries a
// request-scoped transaction to that transaction. It is the bridge between the
// HTTP middleware (which starts a transaction and sets app.tenant_id) and the
// long-lived *gorm.DB handles captured by repositories and services.
//
// Nested transactions must be started with WithContext(ctx, db) so GORM uses a
// savepoint on the request transaction instead of a new connection.
func EnableTenantBinding(db *gorm.DB) error {
	if db == nil {
		return errors.New("database tidak tersedia")
	}
	if _, loaded := boundDatabases.LoadOrStore(db, struct{}{}); loaded {
		return nil
	}

	bind := func(tx *gorm.DB) {
		if tx == nil || tx.Statement == nil {
			return
		}
		requestTx := TxFromContext(tx.Statement.Context)
		if requestTx == nil || requestTx.Statement == nil || requestTx.Statement.ConnPool == nil {
			return
		}
		if _, alreadyBound := tx.Statement.ConnPool.(*sql.Tx); alreadyBound {
			return
		}
		tx.Statement.ConnPool = requestTx.Statement.ConnPool
	}

	// GORM's processor type is unexported, so chain Before(...).Register(...)
	// directly instead of storing the processor in a variable.
	registrations := []func() error{func() error {
		return db.Callback().Create().Before("gorm:create").Register("fixphone:tenant_bind:create", bind)
	},
		func() error {
			return db.Callback().Query().Before("gorm:query").Register("fixphone:tenant_bind:query", bind)
		},
		func() error {
			return db.Callback().Update().Before("gorm:update").Register("fixphone:tenant_bind:update", bind)
		},
		func() error {
			return db.Callback().Delete().Before("gorm:delete").Register("fixphone:tenant_bind:delete", bind)
		},
		func() error {
			return db.Callback().Row().Before("gorm:row").Register("fixphone:tenant_bind:row", bind)
		},
		func() error {
			return db.Callback().Raw().Before("gorm:raw").Register("fixphone:tenant_bind:raw", bind)
		},
	}

	for _, register := range registrations {
		if err := register(); err != nil {
			boundDatabases.Delete(db)
			return err
		}
	}
	return nil
}
