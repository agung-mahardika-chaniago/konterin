package database

import (
	"context"

	"gorm.io/gorm"
)

type txContextKey struct{}

// ContextWithTx attaches a request-scoped transaction to ctx. Repositories keep
// using their shared *gorm.DB handle; EnableTenantBinding rewires every query
// executed with this context to the transaction.
func ContextWithTx(ctx context.Context, tx *gorm.DB) context.Context {
	if ctx == nil || tx == nil {
		return ctx
	}
	return context.WithValue(ctx, txContextKey{}, tx)
}

// TxFromContext returns the request transaction, when present.
func TxFromContext(ctx context.Context) *gorm.DB {
	if ctx == nil {
		return nil
	}
	tx, _ := ctx.Value(txContextKey{}).(*gorm.DB)
	return tx
}

// WithContext returns the request transaction bound to ctx when one exists.
// Service code should use it before starting a transaction or savepoint so
// tenant-scoped work stays inside the request transaction.
func WithContext(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx := TxFromContext(ctx); tx != nil {
		return tx.WithContext(ctx)
	}
	if db == nil {
		return nil
	}
	return db.WithContext(ctx)
}
