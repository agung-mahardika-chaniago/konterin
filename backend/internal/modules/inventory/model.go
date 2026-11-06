// Package inventory implements sparepart master data and stock movements.
package inventory

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

// Sparepart is the API representation of a master part.
type Sparepart struct {
	ID               string     `json:"id"`
	SKU              string     `json:"sku"`
	Barcode          *string    `json:"barcode"`
	Name             string     `json:"name"`
	CategoryID       *string    `json:"category_id"`
	CategoryName     *string    `json:"category_name"`
	SupplierID       *string    `json:"supplier_id"`
	SupplierName     *string    `json:"supplier_name"`
	CompatibleModels *string    `json:"compatible_models"`
	Stock            int        `json:"stock"`
	MinStock         int        `json:"min_stock"`
	CostPrice        int64      `json:"cost_price"`
	SellPrice        int64      `json:"sell_price"`
	StockValue       int64      `json:"stock_value"`
	Location         *string    `json:"location"`
	IsActive         bool       `json:"is_active"`
	LowStock         bool       `json:"low_stock"`
	LastStockAt      *time.Time `json:"last_stock_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// SparepartList is a paginated sparepart result.
type SparepartList struct {
	Items []Sparepart
	Total int64
}

// SparepartFilter is the list query.
type SparepartFilter struct {
	Query      string
	CategoryID *uuid.UUID
	SupplierID *uuid.UUID
	LowStock   bool
	Active     *bool
	Page       int
	PerPage    int
}

// CreateSparepartRequest creates a master part and its optional opening stock.
type CreateSparepartRequest struct {
	SKU              string  `json:"sku" validate:"required,max=40"`
	Barcode          *string `json:"barcode" validate:"omitempty,max=60"`
	Name             string  `json:"name" validate:"required,max=150"`
	CategoryID       *string `json:"category_id"`
	SupplierID       *string `json:"supplier_id"`
	CompatibleModels *string `json:"compatible_models" validate:"omitempty,max=2000"`
	Stock            int     `json:"stock" validate:"gte=0,lte=10000000"`
	MinStock         int     `json:"min_stock" validate:"gte=0,lte=10000000"`
	CostPrice        int64   `json:"cost_price" validate:"gte=0"`
	SellPrice        int64   `json:"sell_price" validate:"gte=0"`
	Location         *string `json:"location" validate:"omitempty,max=60"`
	IsActive         *bool   `json:"is_active"`
}

// UpdateSparepartRequest edits master data. Stock is only changed through
// purchases or stock adjustments.
type UpdateSparepartRequest struct {
	SKU              *string `json:"sku" validate:"omitempty,max=40"`
	Barcode          *string `json:"barcode" validate:"omitempty,max=60"`
	Name             *string `json:"name" validate:"omitempty,max=150"`
	CategoryID       *string `json:"category_id"`
	SupplierID       *string `json:"supplier_id"`
	CompatibleModels *string `json:"compatible_models" validate:"omitempty,max=2000"`
	MinStock         *int    `json:"min_stock" validate:"omitempty,gte=0,lte=10000000"`
	CostPrice        *int64  `json:"cost_price" validate:"omitempty,gte=0"`
	SellPrice        *int64  `json:"sell_price" validate:"omitempty,gte=0"`
	Location         *string `json:"location" validate:"omitempty,max=60"`
	IsActive         *bool   `json:"is_active"`
}

// Category is a sparepart category.
type Category struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	SparepartCount int64     `json:"sparepart_count"`
	CreatedAt      time.Time `json:"created_at"`
}

// CreateCategoryRequest creates a category.
type CreateCategoryRequest struct {
	Name string `json:"name" validate:"required,max=80"`
}

// UpdateCategoryRequest renames a category.
type UpdateCategoryRequest struct {
	Name *string `json:"name" validate:"omitempty,max=80"`
}

// Supplier is a sparepart supplier.
type Supplier struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ContactName *string   `json:"contact_name"`
	Phone       *string   `json:"phone"`
	Address     *string   `json:"address"`
	Notes       *string   `json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateSupplierRequest creates a supplier.
type CreateSupplierRequest struct {
	Name        string  `json:"name" validate:"required,max=120"`
	ContactName *string `json:"contact_name" validate:"omitempty,max=120"`
	Phone       *string `json:"phone" validate:"omitempty,max=30"`
	Address     *string `json:"address" validate:"omitempty,max=500"`
	Notes       *string `json:"notes" validate:"omitempty,max=1000"`
}

// UpdateSupplierRequest edits a supplier.
type UpdateSupplierRequest struct {
	Name        *string `json:"name" validate:"omitempty,max=120"`
	ContactName *string `json:"contact_name" validate:"omitempty,max=120"`
	Phone       *string `json:"phone" validate:"omitempty,max=30"`
	Address     *string `json:"address" validate:"omitempty,max=500"`
	Notes       *string `json:"notes" validate:"omitempty,max=1000"`
}

// PurchaseItemRequest is one line in a stock purchase.
type PurchaseItemRequest struct {
	SparepartID string `json:"sparepart_id" validate:"required,uuid"`
	Qty         int    `json:"qty" validate:"required,gt=0,lte=1000000"`
	UnitCost    int64  `json:"unit_cost" validate:"gte=0"`
}

// PurchaseRequest records stock received from a supplier.
type PurchaseRequest struct {
	SupplierID  *string               `json:"supplier_id"`
	ReferenceNo *string               `json:"reference_no" validate:"omitempty,max=60"`
	PurchasedAt *string               `json:"purchased_at"`
	Items       []PurchaseItemRequest `json:"items" validate:"required,min=1,dive"`
	Notes       *string               `json:"notes" validate:"omitempty,max=1000"`
}

// PurchaseResult is returned after a successful purchase.
type PurchaseResult struct {
	ReferenceID string     `json:"reference_id"`
	PurchasedAt time.Time  `json:"purchased_at"`
	TotalCost   int64      `json:"total_cost"`
	Items       []Movement `json:"items"`
}

// AdjustmentRequest sets the physical stock count for one part.
type AdjustmentRequest struct {
	SparepartID string  `json:"sparepart_id" validate:"required,uuid"`
	NewStock    int     `json:"new_stock" validate:"gte=0,lte=10000000"`
	Reason      string  `json:"reason" validate:"required,max=500"`
	Notes       *string `json:"notes" validate:"omitempty,max=1000"`
}

// Movement is one append-only stock movement.
type Movement struct {
	ID            string    `json:"id"`
	SparepartID   string    `json:"sparepart_id"`
	SparepartName string    `json:"sparepart_name"`
	SKU           string    `json:"sku"`
	OutletID      string    `json:"outlet_id"`
	OutletName    string    `json:"outlet_name"`
	MovementType  string    `json:"movement_type"`
	Qty           int       `json:"qty"`
	StockBefore   int       `json:"stock_before"`
	StockAfter    int       `json:"stock_after"`
	UnitCost      int64     `json:"unit_cost"`
	ReferenceType string    `json:"reference_type"`
	ReferenceID   *string   `json:"reference_id"`
	Notes         *string   `json:"notes"`
	CreatedBy     *string   `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}

// MovementList is a paginated movement result.
type MovementList struct {
	Items []Movement
	Total int64
}

// MovementFilter is the movement list query.
type MovementFilter struct {
	SparepartID  *uuid.UUID
	MovementType string
	OutletID     *uuid.UUID
	FromDate     string
	ToDate       string
	Page         int
	PerPage      int
}

// StockAlert is a part at or below its minimum stock.
type StockAlert struct {
	ID           string  `json:"id"`
	SKU          string  `json:"sku"`
	Name         string  `json:"name"`
	CategoryName *string `json:"category_name"`
	Stock        int     `json:"stock"`
	MinStock     int     `json:"min_stock"`
	Shortage     int     `json:"shortage"`
}

// Valuation summarises inventory value at cost.
type Valuation struct {
	TotalItems int64 `json:"total_items"`
	TotalStock int64 `json:"total_stock"`
	TotalValue int64 `json:"total_value"`
}

// TopUsed is a part ranked by service usage.
type TopUsed struct {
	SparepartID   string `json:"sparepart_id"`
	SKU           string `json:"sku"`
	Name          string `json:"name"`
	UsedQty       int64  `json:"used_qty"`
	MovementCount int64  `json:"movement_count"`
}
