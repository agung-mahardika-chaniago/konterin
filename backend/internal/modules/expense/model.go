// Package expense implements basic operational expense recording.
package expense

import (
	"time"

	"github.com/google/uuid"
)

// Actor is the authenticated staff member plus their tenant/outlet boundary.
type Actor struct {
	UserID          uuid.UUID
	TenantID        uuid.UUID
	Name            string
	Role            string
	Permissions     map[string]struct{}
	AllOutletAccess bool
	OutletIDs       []uuid.UUID
	ActiveOutletID  *uuid.UUID
}

// HasPermission reports whether the actor holds a permission code.
func (a Actor) HasPermission(code string) bool {
	if a.Permissions == nil {
		return false
	}
	_, ok := a.Permissions[code]
	return ok
}

// HasOutlet reports whether the actor may use the outlet.
func (a Actor) HasOutlet(outletID uuid.UUID) bool {
	for _, id := range a.OutletIDs {
		if id == outletID {
			return true
		}
	}
	return false
}

// Category is an expense category.
type Category struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	IsSystem     bool   `json:"is_system"`
	ExpenseCount int64  `json:"expense_count"`
}

// ExpenseListItem is one expense row.
type ExpenseListItem struct {
	ID           string    `json:"id"`
	ExpenseNo    string    `json:"expense_no"`
	CategoryID   string    `json:"category_id"`
	CategoryName string    `json:"category_name"`
	OutletID     string    `json:"outlet_id"`
	OutletName   string    `json:"outlet_name"`
	Description  string    `json:"description"`
	Amount       int64     `json:"amount"`
	ExpenseDate  string    `json:"expense_date"`
	Method       string    `json:"method"`
	ReceiptPath  *string   `json:"receipt_path"`
	Notes        *string   `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ExpenseList is a paginated expense result plus the filtered total.
type ExpenseList struct {
	Items       []ExpenseListItem
	Total       int64
	TotalAmount int64
}

// ExpenseFilter is the expense list query.
type ExpenseFilter struct {
	Query      string
	CategoryID uuid.UUID
	OutletID   uuid.UUID
	FromDate   string
	ToDate     string
	Page       int
	PerPage    int
}

// CreateExpenseRequest records a new expense.
type CreateExpenseRequest struct {
	CategoryID  string  `json:"category_id" validate:"required"`
	Description string  `json:"description" validate:"required,max=200"`
	Amount      int64   `json:"amount"`
	ExpenseDate string  `json:"expense_date" validate:"required"`
	Method      string  `json:"method" validate:"required,oneof=cash bank_transfer qris other"`
	Notes       *string `json:"notes" validate:"omitempty,max=1000"`
}

// UpdateExpenseRequest edits an existing expense.
type UpdateExpenseRequest struct {
	CategoryID  *string `json:"category_id"`
	Description *string `json:"description" validate:"omitempty,max=200"`
	Amount      *int64  `json:"amount"`
	ExpenseDate *string `json:"expense_date"`
	Method      *string `json:"method" validate:"omitempty,oneof=cash bank_transfer qris other"`
	Notes       *string `json:"notes" validate:"omitempty,max=1000"`
}

// CreateCategoryRequest adds a custom expense category.
type CreateCategoryRequest struct {
	Name string `json:"name" validate:"required,max=80"`
}
