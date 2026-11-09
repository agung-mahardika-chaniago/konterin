package tracking

import (
	"context"
	"strings"
	"time"

	"fixphone/backend/internal/modules/notification"
	"fixphone/backend/internal/pkg/apperror"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service resolves public tracking data.
type Service struct {
	db *gorm.DB
}

// NewService builds the tracking module.
func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type trackingRow struct {
	ID                 uuid.UUID  `gorm:"column:id"`
	TenantID           uuid.UUID  `gorm:"column:tenant_id"`
	OutletID           uuid.UUID  `gorm:"column:outlet_id"`
	TicketNo           string     `gorm:"column:ticket_no"`
	Status             string     `gorm:"column:status"`
	ReceivedAt         time.Time  `gorm:"column:received_at"`
	EstimatedDoneAt    *time.Time `gorm:"column:estimated_done_at"`
	DeviceBrand        string     `gorm:"column:device_brand"`
	DeviceModel        string     `gorm:"column:device_model"`
	Complaint          string     `gorm:"column:complaint"`
	PickedUpAt         *time.Time `gorm:"column:picked_up_at"`
	PublicTokenExpires *time.Time `gorm:"column:public_token_expires_at"`
	StoreName          string     `gorm:"column:store_name"`
	StoreLogo          *string    `gorm:"column:logo_path"`
}

const trackingSelect = `
	SELECT so.id, so.tenant_id, so.outlet_id, so.ticket_no, so.status,
	       so.received_at, so.estimated_done_at, so.device_brand, so.device_model,
	       so.complaint, so.picked_up_at, so.public_token_expires_at,
	       bs.store_name, bs.logo_path
	FROM service_orders so
	JOIN customers c ON c.id = so.customer_id
	JOIN LATERAL (
		SELECT store_name, logo_path, public_tracking_enabled
		FROM business_settings b
		WHERE b.tenant_id = so.tenant_id
		  AND (b.outlet_id = so.outlet_id OR b.outlet_id IS NULL)
		ORDER BY (b.outlet_id IS NULL)
		LIMIT 1
	) bs ON true
`

// FindByTicket looks up an order with ticket + last four phone digits. Both
// values are required; failures return one generic message on purpose.
func (s *Service) FindByTicket(ctx context.Context, ticket, phoneLast4 string) (*PublicTracking, error) {
	ticket = strings.ToUpper(strings.TrimSpace(ticket))
	phoneLast4 = strings.TrimSpace(phoneLast4)
	if ticket == "" || len(phoneLast4) != 4 || !isDigits(phoneLast4) {
		return nil, apperror.Validation(map[string]string{
			"ticket":       "Nomor tiket wajib diisi.",
			"phone_last4":  "Empat digit terakhir nomor HP wajib berupa angka.",
		})
	}
	var row trackingRow
	if err := s.db.WithContext(ctx).Raw(trackingSelect+`
		WHERE UPPER(so.ticket_no) = ? AND RIGHT(c.phone_normalized, 4) = ?
		  AND bs.public_tracking_enabled = true
		LIMIT 1`, ticket, phoneLast4).Scan(&row).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if row.ID == uuid.Nil {
		return nil, notFound()
	}
	return s.build(ctx, row)
}

// FindByToken resolves the QR/deep-link token and enforces its expiry.
func (s *Service) FindByToken(ctx context.Context, token string) (*PublicTracking, error) {
	token = strings.TrimSpace(token)
	if len(token) != 32 || !isHex(token) {
		return nil, notFound()
	}
	var row trackingRow
	if err := s.db.WithContext(ctx).Raw(trackingSelect+`
		WHERE so.public_token = ? AND so.public_token_expires_at > now()
		  AND bs.public_tracking_enabled = true
		LIMIT 1`, token).Scan(&row).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if row.ID == uuid.Nil {
		return nil, notFound()
	}
	return s.build(ctx, row)
}

// StoreByToken returns the store identity for a valid public token.
func (s *Service) StoreByToken(ctx context.Context, token string) (StoreInfo, error) {
	token = strings.TrimSpace(token)
	if len(token) != 32 || !isHex(token) {
		return StoreInfo{}, notFound()
	}
	var row struct {
		StoreName string  `gorm:"column:store_name"`
		LogoPath  *string `gorm:"column:logo_path"`
	}
	if err := s.db.WithContext(ctx).Raw(trackingSelect+`
		WHERE so.public_token = ? AND so.public_token_expires_at > now()
		  AND bs.public_tracking_enabled = true
		LIMIT 1`, token).Scan(&row).Error; err != nil {
		return StoreInfo{}, apperror.Internal(err)
	}
	if strings.TrimSpace(row.StoreName) == "" {
		return StoreInfo{}, notFound()
	}
	return StoreInfo{Name: row.StoreName, LogoURL: row.LogoPath}, nil
}

func (s *Service) build(ctx context.Context, row trackingRow) (*PublicTracking, error) {
	out := &PublicTracking{
		TicketNo:           row.TicketNo,
		Store:              StoreInfo{Name: row.StoreName, LogoURL: row.StoreLogo},
		Status:             row.Status,
		StatusLabel:        notification.StatusLabel(row.Status),
		ReceivedAt:         row.ReceivedAt,
		EstimatedDoneAt:    row.EstimatedDoneAt,
		Device:             strings.TrimSpace(row.DeviceBrand + " " + row.DeviceModel),
		Complaint:          row.Complaint,
		PaymentStatusLabel: "Tanpa Tagihan",
		PickupStatus:       "Belum Diambil",
		Warranty:           WarrantyInfo{Status: "none"},
	}
	if row.PickedUpAt != nil {
		out.PickupStatus = "Sudah Diambil"
	}

	var invoice struct {
		Status          string `gorm:"column:status"`
		PaymentStatus   string `gorm:"column:payment_status"`
		TotalAmount     int64  `gorm:"column:total_amount"`
		RemainingAmount int64  `gorm:"column:remaining_amount"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT status, total_amount, remaining_amount
		FROM invoices WHERE tenant_id = ? AND service_order_id = ?`,
		row.TenantID, row.ID).Scan(&invoice).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if strings.TrimSpace(invoice.Status) != "" {
		status := invoice.Status
		out.InvoiceStatus = &status
		switch invoice.Status {
		case "paid":
			out.PaymentStatusLabel = "Lunas"
		case "partial":
			out.PaymentStatusLabel = "Belum Lunas"
		case "unpaid":
			out.PaymentStatusLabel = "Belum Lunas"
		case "draft":
			out.PaymentStatusLabel = "Menunggu Finalisasi"
		case "void":
			out.PaymentStatusLabel = "Dibatalkan"
		}
		if invoice.Status != "void" && invoice.Status != "draft" && invoice.RemainingAmount == 0 {
			out.PaymentStatusLabel = "Lunas"
		}
	}

	var diagnosis string
	if err := s.db.WithContext(ctx).Raw(`
		SELECT COALESCE(diagnosis, '') FROM service_diagnostics
		WHERE tenant_id = ? AND service_order_id = ?`,
		row.TenantID, row.ID).Scan(&diagnosis).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if strings.TrimSpace(diagnosis) != "" {
		value := strings.TrimSpace(diagnosis)
		out.Diagnosis = &value
	}

	var warranty struct {
		Status  string    `gorm:"column:status"`
		Days    int       `gorm:"column:days"`
		EndDate time.Time `gorm:"column:end_date"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT status, days, end_date FROM warranties
		WHERE tenant_id = ? AND service_order_id = ?`,
		row.TenantID, row.ID).Scan(&warranty).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if strings.TrimSpace(warranty.Status) != "" {
		end := warranty.EndDate.Format("2006-01-02")
		out.Warranty = WarrantyInfo{
			Active:  warranty.Status == "active" && !warranty.EndDate.Before(dateOnly(time.Now())),
			Status:  warranty.Status,
			Days:    warranty.Days,
			EndDate: &end,
		}
	}
	return out, nil
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func notFound() error {
	return apperror.NotFound("Data servis tidak ditemukan. Periksa nomor tiket dan 4 digit terakhir nomor HP.")
}

func isDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isHex(value string) bool {
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}
