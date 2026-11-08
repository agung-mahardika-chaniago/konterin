package expense

import (
	"context"
	"errors"
	"strings"
	"time"

	"fixphone/backend/internal/modules/audit"
	"fixphone/backend/internal/pkg/apperror"
	"fixphone/backend/internal/pkg/docnum"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var allowedExpenseMethods = map[string]struct{}{
	"cash": {}, "bank_transfer": {}, "qris": {}, "other": {},
}

// Service implements expense and category operations.
type Service struct {
	db *gorm.DB
}

// NewService builds the expense module.
func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func actorTenant(actor Actor) (uuid.UUID, error) {
	if actor.TenantID == uuid.Nil {
		return uuid.Nil, apperror.Forbidden("FORBIDDEN", "Endpoint ini hanya tersedia untuk pengguna tenant.")
	}
	return actor.TenantID, nil
}

func actorID(actor Actor) *uuid.UUID {
	id := actor.UserID
	if id == uuid.Nil {
		return nil
	}
	return &id
}

func (a Actor) outletFor(requested *uuid.UUID) (uuid.UUID, error) {
	if requested != nil {
		if !a.HasOutlet(*requested) {
			return uuid.Nil, apperror.Forbidden("OUTLET_FORBIDDEN", "Anda tidak memiliki akses ke outlet ini.")
		}
		return *requested, nil
	}
	if a.ActiveOutletID == nil {
		return uuid.Nil, apperror.Validation(map[string]string{"outlet_id": "Pilih outlet terlebih dahulu."})
	}
	return *a.ActiveOutletID, nil
}

func trimOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func parseExpenseDate(raw string) (time.Time, error) {
	value, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, apperror.Validation(map[string]string{"expense_date": "Format tanggal harus YYYY-MM-DD."})
	}
	return value, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type expenseRow struct {
	ID           uuid.UUID `gorm:"column:id"`
	ExpenseNo    string    `gorm:"column:expense_no"`
	CategoryID   uuid.UUID `gorm:"column:category_id"`
	CategoryName string    `gorm:"column:category_name"`
	OutletID     uuid.UUID `gorm:"column:outlet_id"`
	OutletName   string    `gorm:"column:outlet_name"`
	Description  string    `gorm:"column:description"`
	Amount       int64     `gorm:"column:amount"`
	ExpenseDate  time.Time `gorm:"column:expense_date"`
	Method       string    `gorm:"column:method"`
	ReceiptPath  *string   `gorm:"column:receipt_path"`
	Notes        *string   `gorm:"column:notes"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (row expenseRow) toItem() ExpenseListItem {
	return ExpenseListItem{
		ID: row.ID.String(), ExpenseNo: row.ExpenseNo,
		CategoryID: row.CategoryID.String(), CategoryName: row.CategoryName,
		OutletID: row.OutletID.String(), OutletName: row.OutletName,
		Description: row.Description, Amount: row.Amount,
		ExpenseDate: row.ExpenseDate.Format("2006-01-02"),
		Method:      row.Method, ReceiptPath: row.ReceiptPath, Notes: row.Notes,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

const expenseBaseQuery = `
	FROM expenses e
	JOIN expense_categories ec ON ec.id = e.category_id
	JOIN outlets o ON o.id = e.outlet_id`

const expenseSelectColumns = `
	e.id, e.expense_no, e.category_id, ec.name AS category_name,
	e.outlet_id, o.name AS outlet_name, e.description, e.amount,
	e.expense_date, e.method, e.receipt_path, e.notes, e.created_at, e.updated_at`

// ListExpenses returns expenses for the active outlet and filtered period.
func (s *Service) ListExpenses(ctx context.Context, actor Actor, filter ExpenseFilter) (*ExpenseList, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	var requested *uuid.UUID
	if filter.OutletID != uuid.Nil {
		id := filter.OutletID
		requested = &id
	}
	outletID, err := actor.outletFor(requested)
	if err != nil {
		return nil, err
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 100 {
		filter.PerPage = 20
	}

	where := ` WHERE e.tenant_id = ? AND e.outlet_id = ? AND e.deleted_at IS NULL`
	args := []any{tenantID, outletID}
	if query := strings.TrimSpace(filter.Query); query != "" {
		like := "%" + query + "%"
		where += ` AND (e.expense_no ILIKE ? OR e.description ILIKE ? OR ec.name ILIKE ?)`
		args = append(args, like, like, like)
	}
	if filter.CategoryID != uuid.Nil {
		where += ` AND e.category_id = ?`
		args = append(args, filter.CategoryID)
	}
	if from := strings.TrimSpace(filter.FromDate); from != "" {
		value, parseErr := parseExpenseDate(from)
		if parseErr != nil {
			return nil, apperror.Validation(map[string]string{"from": "Format tanggal awal harus YYYY-MM-DD."})
		}
		where += ` AND e.expense_date >= ?`
		args = append(args, value.Format("2006-01-02"))
	}
	if to := strings.TrimSpace(filter.ToDate); to != "" {
		value, parseErr := parseExpenseDate(to)
		if parseErr != nil {
			return nil, apperror.Validation(map[string]string{"to": "Format tanggal akhir harus YYYY-MM-DD."})
		}
		where += ` AND e.expense_date <= ?`
		args = append(args, value.Format("2006-01-02"))
	}

	var totals struct {
		Total       int64 `gorm:"column:total"`
		TotalAmount int64 `gorm:"column:total_amount"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT count(*) AS total, COALESCE(SUM(e.amount), 0) AS total_amount`+
		expenseBaseQuery+where, args...).Scan(&totals).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	queryArgs := append(append([]any{}, args...), filter.PerPage, (filter.Page-1)*filter.PerPage)
	var rows []expenseRow
	if err := s.db.WithContext(ctx).Raw(`
		SELECT `+expenseSelectColumns+expenseBaseQuery+where+`
		ORDER BY e.expense_date DESC, e.created_at DESC
		LIMIT ? OFFSET ?`, queryArgs...).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	items := make([]ExpenseListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toItem())
	}
	return &ExpenseList{Items: items, Total: totals.Total, TotalAmount: totals.TotalAmount}, nil
}

func (s *Service) categoryExists(ctx context.Context, db *gorm.DB, tenantID, categoryID uuid.UUID) (bool, error) {
	var exists bool
	if err := db.WithContext(ctx).Raw(`
		SELECT EXISTS (
			SELECT 1 FROM expense_categories WHERE id = ? AND tenant_id = ?
		)`, categoryID, tenantID).Scan(&exists).Error; err != nil {
		return false, err
	}
	return exists, nil
}

// CreateExpense records a new expense and its document number.
func (s *Service) CreateExpense(ctx context.Context, actor Actor, input CreateExpenseRequest) (*ExpenseListItem, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	outletID, err := actor.outletFor(nil)
	if err != nil {
		return nil, err
	}
	categoryID, parseErr := uuid.Parse(strings.TrimSpace(input.CategoryID))
	if parseErr != nil {
		return nil, apperror.Validation(map[string]string{"category_id": "ID kategori tidak valid."})
	}
	exists, err := s.categoryExists(ctx, s.db, tenantID, categoryID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if !exists {
		return nil, apperror.Validation(map[string]string{"category_id": "Kategori pengeluaran tidak ditemukan."})
	}
	description := strings.TrimSpace(input.Description)
	if description == "" {
		return nil, apperror.Validation(map[string]string{"description": "Deskripsi pengeluaran wajib diisi."})
	}
	if input.Amount <= 0 {
		return nil, apperror.Validation(map[string]string{"amount": "Nominal pengeluaran harus lebih dari 0."})
	}
	method := strings.ToLower(strings.TrimSpace(input.Method))
	if _, ok := allowedExpenseMethods[method]; !ok {
		return nil, apperror.Validation(map[string]string{"method": "Metode pengeluaran tidak dikenal."})
	}
	expenseDate, err := parseExpenseDate(input.ExpenseDate)
	if err != nil {
		return nil, err
	}
	notes := trimOptional(input.Notes)

	var expenseID uuid.UUID
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		number, numberErr := docnum.Next(ctx, tx, tenantID, outletID, "expense",
			"EXP", "{PREFIX}-{YYMMDD}-{SEQ:4}", now)
		if numberErr != nil {
			return apperror.Internal(numberErr)
		}
		var idErr error
		expenseID, idErr = uuid.NewV7()
		if idErr != nil {
			return apperror.Internal(idErr)
		}
		if err := tx.Exec(`
			INSERT INTO expenses (
				id, tenant_id, outlet_id, expense_no, category_id, description,
				amount, expense_date, method, notes, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			expenseID, tenantID, outletID, number, categoryID, description,
			input.Amount, expenseDate.Format("2006-01-02"), method, notes, now, now).Error; err != nil {
			return apperror.Internal(err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
			Action: "expense_create", Module: "expense",
			EntityType: "expense", EntityID: &expenseID,
			After: map[string]any{
				"expense_no": number, "category_id": categoryID,
				"amount": input.Amount, "expense_date": input.ExpenseDate,
				"method": method,
			},
		})
	})
	if err != nil {
		if _, ok := apperror.As(err); ok {
			return nil, err
		}
		return nil, apperror.Internal(err)
	}
	return s.getExpense(ctx, tenantID, outletID, expenseID)
}

func (s *Service) getExpense(ctx context.Context, tenantID, outletID, expenseID uuid.UUID) (*ExpenseListItem, error) {
	var row expenseRow
	if err := s.db.WithContext(ctx).Raw(`
		SELECT `+expenseSelectColumns+expenseBaseQuery+`
		WHERE e.id = ? AND e.tenant_id = ? AND e.outlet_id = ? AND e.deleted_at IS NULL`,
		expenseID, tenantID, outletID).Scan(&row).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if row.ID == uuid.Nil {
		return nil, apperror.NotFound("Pengeluaran tidak ditemukan.")
	}
	item := row.toItem()
	return &item, nil
}

// UpdateExpense edits an expense.
func (s *Service) UpdateExpense(ctx context.Context, actor Actor, expenseID uuid.UUID, input UpdateExpenseRequest) (*ExpenseListItem, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current expenseRow
		if err := tx.Raw(`
			SELECT e.id, e.expense_no, e.category_id, e.outlet_id, e.description,
			       e.amount, e.expense_date, e.method, e.notes
			FROM expenses e
			WHERE e.id = ? AND e.tenant_id = ? AND e.deleted_at IS NULL
			FOR UPDATE`, expenseID, tenantID).Scan(&current).Error; err != nil {
			return apperror.Internal(err)
		}
		if current.ID == uuid.Nil || !actor.HasOutlet(current.OutletID) {
			return apperror.NotFound("Pengeluaran tidak ditemukan.")
		}

		categoryID := current.CategoryID
		if input.CategoryID != nil {
			parsed, parseErr := uuid.Parse(strings.TrimSpace(*input.CategoryID))
			if parseErr != nil {
				return apperror.Validation(map[string]string{"category_id": "ID kategori tidak valid."})
			}
			exists, checkErr := s.categoryExists(ctx, tx, tenantID, parsed)
			if checkErr != nil {
				return apperror.Internal(checkErr)
			}
			if !exists {
				return apperror.Validation(map[string]string{"category_id": "Kategori pengeluaran tidak ditemukan."})
			}
			categoryID = parsed
		}
		description := current.Description
		if input.Description != nil {
			description = strings.TrimSpace(*input.Description)
			if description == "" {
				return apperror.Validation(map[string]string{"description": "Deskripsi pengeluaran wajib diisi."})
			}
		}
		amount := current.Amount
		if input.Amount != nil {
			if *input.Amount <= 0 {
				return apperror.Validation(map[string]string{"amount": "Nominal pengeluaran harus lebih dari 0."})
			}
			amount = *input.Amount
		}
		method := current.Method
		if input.Method != nil {
			method = strings.ToLower(strings.TrimSpace(*input.Method))
			if _, ok := allowedExpenseMethods[method]; !ok {
				return apperror.Validation(map[string]string{"method": "Metode pengeluaran tidak dikenal."})
			}
		}
		expenseDate := current.ExpenseDate
		if input.ExpenseDate != nil {
			parsed, parseErr := parseExpenseDate(*input.ExpenseDate)
			if parseErr != nil {
				return parseErr
			}
			expenseDate = parsed
		}
		notes := current.Notes
		if input.Notes != nil {
			notes = trimOptional(input.Notes)
		}

		now := time.Now().UTC()
		if err := tx.Exec(`
			UPDATE expenses SET category_id = ?, description = ?, amount = ?,
				expense_date = ?, method = ?, notes = ?, updated_at = ?
			WHERE id = ? AND tenant_id = ?`,
			categoryID, description, amount, expenseDate.Format("2006-01-02"), method, notes, now,
			expenseID, tenantID).Error; err != nil {
			return apperror.Internal(err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
			Action: "expense_update", Module: "expense",
			EntityType: "expense", EntityID: &expenseID,
			Before: map[string]any{
				"category_id": current.CategoryID, "description": current.Description,
				"amount": current.Amount, "expense_date": current.ExpenseDate.Format("2006-01-02"),
				"method": current.Method, "notes": current.Notes,
			},
			After: map[string]any{
				"category_id": categoryID, "description": description,
				"amount": amount, "expense_date": expenseDate.Format("2006-01-02"),
				"method": method, "notes": notes,
			},
		})
	})
	if err != nil {
		if _, ok := apperror.As(err); ok {
			return nil, err
		}
		return nil, apperror.Internal(err)
	}

	var row struct {
		OutletID uuid.UUID `gorm:"column:outlet_id"`
	}
	if err := s.db.WithContext(ctx).Raw(`SELECT outlet_id FROM expenses WHERE id = ? AND tenant_id = ?`,
		expenseID, tenantID).Scan(&row).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	return s.getExpense(ctx, tenantID, row.OutletID, expenseID)
}

// DeleteExpense soft-deletes an expense.
func (s *Service) DeleteExpense(ctx context.Context, actor Actor, expenseID uuid.UUID) error {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current expenseRow
		if err := tx.Raw(`
			SELECT e.id, e.expense_no, e.outlet_id, e.description, e.amount, e.expense_date
			FROM expenses e
			WHERE e.id = ? AND e.tenant_id = ? AND e.deleted_at IS NULL
			FOR UPDATE`, expenseID, tenantID).Scan(&current).Error; err != nil {
			return apperror.Internal(err)
		}
		if current.ID == uuid.Nil || !actor.HasOutlet(current.OutletID) {
			return apperror.NotFound("Pengeluaran tidak ditemukan.")
		}
		now := time.Now().UTC()
		if err := tx.Exec(`
			UPDATE expenses SET deleted_at = ?, updated_at = ?
			WHERE id = ? AND tenant_id = ?`, now, now, expenseID, tenantID).Error; err != nil {
			return apperror.Internal(err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
			Action: "expense_delete", Module: "expense",
			EntityType: "expense", EntityID: &expenseID,
			Before: map[string]any{
				"expense_no": current.ExpenseNo, "description": current.Description,
				"amount": current.Amount, "expense_date": current.ExpenseDate.Format("2006-01-02"),
			},
		})
	})
}

// ListCategories returns expense categories with usage counts.
func (s *Service) ListCategories(ctx context.Context, actor Actor) ([]Category, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID           uuid.UUID `gorm:"column:id"`
		Name         string    `gorm:"column:name"`
		IsSystem     bool      `gorm:"column:is_system"`
		ExpenseCount int64     `gorm:"column:expense_count"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT ec.id, ec.name, ec.is_system,
		       count(e.id) FILTER (WHERE e.deleted_at IS NULL) AS expense_count
		FROM expense_categories ec
		LEFT JOIN expenses e ON e.category_id = ec.id AND e.tenant_id = ec.tenant_id
		WHERE ec.tenant_id = ?
		GROUP BY ec.id, ec.name, ec.is_system
		ORDER BY ec.is_system DESC, ec.name`, tenantID).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	categories := make([]Category, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, Category{
			ID: row.ID.String(), Name: row.Name,
			IsSystem: row.IsSystem, ExpenseCount: row.ExpenseCount,
		})
	}
	return categories, nil
}

// CreateCategory adds a custom expense category.
func (s *Service) CreateCategory(ctx context.Context, actor Actor, input CreateCategoryRequest) (*Category, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, apperror.Validation(map[string]string{"name": "Nama kategori wajib diisi."})
	}
	var existing bool
	if err := s.db.WithContext(ctx).Raw(`
		SELECT EXISTS (
			SELECT 1 FROM expense_categories WHERE tenant_id = ? AND lower(name) = lower(?)
		)`, tenantID, name).Scan(&existing).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if existing {
		return nil, apperror.Conflict("CATEGORY_EXISTS", "Kategori pengeluaran dengan nama tersebut sudah ada.")
	}
	categoryID, idErr := uuid.NewV7()
	if idErr != nil {
		return nil, apperror.Internal(idErr)
	}
	now := time.Now().UTC()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO expense_categories (id, tenant_id, name, is_system, created_at, updated_at)
			VALUES (?, ?, ?, false, ?, ?)`, categoryID, tenantID, name, now, now).Error; err != nil {
			return err
		}
		return audit.Write(ctx, tx, audit.Entry{
			TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
			Action: "expense_category_create", Module: "expense",
			EntityType: "expense_category", EntityID: &categoryID,
			After: map[string]any{"name": name},
		})
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperror.Conflict("CATEGORY_EXISTS", "Kategori pengeluaran dengan nama tersebut sudah ada.")
		}
		if _, ok := apperror.As(err); ok {
			return nil, err
		}
		return nil, apperror.Internal(err)
	}
	return &Category{ID: categoryID.String(), Name: name, IsSystem: false}, nil
}
