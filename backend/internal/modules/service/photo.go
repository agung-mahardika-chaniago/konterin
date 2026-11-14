package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"fixphone/backend/internal/modules/audit"
	"fixphone/backend/internal/pkg/apperror"

	"github.com/google/uuid"
)

var photoStages = map[string]struct{}{
	"intake": {}, "before": {}, "after": {}, "testing": {}, "other": {},
}

var photoExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// AddPhoto validates, stores, and records a service photo.
func (s *Service) AddPhoto(
	ctx context.Context,
	actor Actor,
	orderID uuid.UUID,
	file *multipart.FileHeader,
	stage string,
	isPublic bool,
) (*ServicePhoto, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	if _, err := s.findOrder(ctx, actor, orderID); err != nil {
		return nil, err
	}
	stage = strings.TrimSpace(stage)
	if stage == "" {
		stage = "intake"
	}
	if _, ok := photoStages[stage]; !ok {
		return nil, apperror.Validation(map[string]string{"stage": "Tahap foto tidak valid."})
	}
	if file == nil {
		return nil, apperror.Validation(map[string]string{"file": "Pilih file foto terlebih dahulu."})
	}
	if file.Size <= 0 {
		return nil, apperror.Validation(map[string]string{"file": "File foto kosong."})
	}
	if file.Size > s.opts.UploadMaxBytes {
		return nil, apperror.New(http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE",
			fmt.Sprintf("Ukuran foto maksimal %d MB.", s.opts.UploadMaxBytes/(1<<20)))
	}

	source, err := file.Open()
	if err != nil {
		return nil, apperror.BadRequest("File foto tidak dapat dibaca.")
	}
	defer source.Close()

	head := make([]byte, 512)
	read, err := io.ReadFull(source, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, apperror.BadRequest("File foto tidak dapat dibaca.")
	}
	mimeType := http.DetectContentType(head[:read])
	ext, ok := photoExtensions[mimeType]
	if !ok {
		return nil, apperror.Validation(map[string]string{"file": "Format foto harus JPEG, PNG, atau WebP."})
	}

	photoID, err := uuid.NewV7()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	relativeDir := path.Join("services", orderID.String())
	diskDir := filepath.Join(s.opts.UploadDir, filepath.FromSlash(relativeDir))
	if err := os.MkdirAll(diskDir, 0o750); err != nil {
		return nil, apperror.Internal(err)
	}
	fileName := photoID.String() + ext
	diskPath := filepath.Join(diskDir, fileName)
	destination, err := os.OpenFile(diskPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) // #nosec G304 -- path dibentuk dari folder upload terkonfigurasi dan UUID internal.
	if err != nil {
		return nil, apperror.Internal(err)
	}

	written, copyErr := io.Copy(destination, io.LimitReader(source, s.opts.UploadMaxBytes+1))
	closeErr := destination.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(diskPath)
		return nil, apperror.Internal(errors.Join(copyErr, closeErr))
	}
	if written > s.opts.UploadMaxBytes {
		_ = os.Remove(diskPath)
		return nil, apperror.New(http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE",
			fmt.Sprintf("Ukuran foto maksimal %d MB.", s.opts.UploadMaxBytes/(1<<20)))
	}

	urlPath := "/uploads/" + path.Join(relativeDir, fileName)
	if err := s.db.WithContext(ctx).Exec(`
		INSERT INTO service_photos (
			id, tenant_id, service_order_id, stage, file_path, mime_type, size_bytes, is_public, uploaded_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		photoID, tenantID, orderID, stage, urlPath, mimeType, written, isPublic, actorID(actor)).Error; err != nil {
		_ = os.Remove(diskPath)
		return nil, apperror.Internal(err)
	}

	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
		Action: "service.photo_upload", Module: "service", EntityType: "service_photo", EntityID: &photoID,
		After: map[string]any{"service_order_id": orderID.String(), "stage": stage, "mime_type": mimeType},
	})
	return &ServicePhoto{
		ID: photoID.String(), Stage: stage, FilePath: urlPath, MimeType: mimeType,
		SizeBytes: written, IsPublic: isPublic, CreatedAt: time.Now().UTC(),
	}, nil
}

// DeletePhoto removes a photo record and its file.
func (s *Service) DeletePhoto(ctx context.Context, actor Actor, orderID, photoID uuid.UUID) error {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return err
	}
	if _, err := s.findOrder(ctx, actor, orderID); err != nil {
		return err
	}
	var filePath string
	if err := s.db.WithContext(ctx).Raw(`
		SELECT file_path FROM service_photos
		WHERE id = ? AND service_order_id = ? AND tenant_id = ?`,
		photoID, orderID, tenantID).Scan(&filePath).Error; err != nil {
		return apperror.Internal(err)
	}
	if filePath == "" {
		return apperror.NotFound("Foto tidak ditemukan.")
	}
	if err := s.db.WithContext(ctx).Exec(`
		DELETE FROM service_photos WHERE id = ? AND service_order_id = ? AND tenant_id = ?`,
		photoID, orderID, tenantID).Error; err != nil {
		return apperror.Internal(err)
	}

	relative := strings.TrimPrefix(filePath, "/uploads/")
	diskPath := filepath.Join(s.opts.UploadDir, filepath.FromSlash(relative))
	if strings.HasPrefix(diskPath, filepath.Clean(s.opts.UploadDir)+string(os.PathSeparator)) {
		_ = os.Remove(diskPath)
	}

	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
		Action: "service.photo_delete", Module: "service", EntityType: "service_photo", EntityID: &photoID,
		Before: map[string]any{"service_order_id": orderID.String(), "file_path": filePath},
	})
	return nil
}
