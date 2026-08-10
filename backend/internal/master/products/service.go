package products

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"erp-system/backend/internal/audit"
	"erp-system/backend/internal/auth"
)

type Service struct {
	repo     *Repository
	auditSvc *audit.Service
}

func NewService(repo *Repository, auditSvc *audit.Service) *Service {
	return &Service{repo: repo, auditSvc: auditSvc}
}

var (
	ErrProductNotFound         = errors.New("product not found")
	ErrProductDeleted          = errors.New("product deleted")
	ErrProductDuplicateSKU     = errors.New("sku already exists")
	ErrProductDuplicateBarcode = errors.New("barcode already exists")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Product, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}
	return product, nil
}

func (s *Service) List(ctx context.Context, filter ProductFilter) ([]Product, error) {
	return s.repo.List(ctx, filter)
}

func (s *Service) Create(ctx context.Context, product *Product) (int64, error) {
	if err := validateProduct(product); err != nil {
		return 0, err
	}
	normalizeOptionalStrings(product)

	if existing, err := s.repo.GetBySKU(ctx, product.SKU); err == nil && existing != nil {
		return 0, ErrProductDuplicateSKU
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	if product.Barcode != nil {
		if existing, err := s.repo.GetByBarcode(ctx, *product.Barcode); err == nil && existing != nil {
			return 0, ErrProductDuplicateBarcode
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	id, err := s.repo.CreateWithTx(ctx, tx, product)
	if err != nil {
		return 0, err
	}

	if s.auditSvc != nil {
		actorUserID := actorUserIDFromContext(ctx)
		resourceID := fmt.Sprintf("%d", id)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserID,
			Action:      "product.create",
			Resource:    "product",
			ResourceID:  &resourceID,
			Metadata: map[string]any{
				"sku": product.SKU,
			},
		})
		if err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return id, nil
}

func (s *Service) Update(ctx context.Context, product *Product) error {
	existing, err := s.repo.GetByIDIncludeDeleted(ctx, product.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrProductNotFound
		}
		return err
	}
	if existing.DeletedAt != nil {
		return ErrProductDeleted
	}

	if err := validateProduct(product); err != nil {
		return err
	}
	normalizeOptionalStrings(product)

	if existingSKU, err := s.repo.GetBySKU(ctx, product.SKU); err == nil && existingSKU != nil && existingSKU.ID != product.ID {
		return ErrProductDuplicateSKU
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if product.Barcode != nil {
		if existingBarcode, err := s.repo.GetByBarcode(ctx, *product.Barcode); err == nil && existingBarcode != nil && existingBarcode.ID != product.ID {
			return ErrProductDuplicateBarcode
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err := s.repo.UpdateWithTx(ctx, tx, product); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrProductNotFound
		}
		return err
	}

	if s.auditSvc != nil {
		actorUserID := actorUserIDFromContext(ctx)
		resourceID := fmt.Sprintf("%d", product.ID)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserID,
			Action:      "product.update",
			Resource:    "product",
			ResourceID:  &resourceID,
			Metadata: map[string]any{
				"sku": product.SKU,
			},
		})
		if err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *Service) SoftDelete(ctx context.Context, id int64) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrProductNotFound
		}
		return err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err := s.repo.SoftDeleteWithTx(ctx, tx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrProductNotFound
		}
		return err
	}

	if s.auditSvc != nil {
		actorUserID := actorUserIDFromContext(ctx)
		resourceID := fmt.Sprintf("%d", id)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserID,
			Action:      "product.delete",
			Resource:    "product",
			ResourceID:  &resourceID,
			Metadata:    map[string]any{},
		})
		if err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func validateProduct(product *Product) error {
	if strings.TrimSpace(product.SKU) == "" {
		return &ValidationError{Field: "sku", Message: "sku is required"}
	}
	if strings.TrimSpace(product.Name) == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	if product.PurchasePrice < 0 {
		return &ValidationError{Field: "purchase_price", Message: "purchase_price must be greater than or equal to 0"}
	}
	if product.SellingPrice < 0 {
		return &ValidationError{Field: "selling_price", Message: "selling_price must be greater than or equal to 0"}
	}
	return nil
}

func normalizeOptionalStrings(product *Product) {
	if product.Barcode != nil {
		trimmed := strings.TrimSpace(*product.Barcode)
		if trimmed == "" {
			product.Barcode = nil
		} else {
			product.Barcode = &trimmed
		}
	}
	if product.Description != nil {
		trimmed := strings.TrimSpace(*product.Description)
		if trimmed == "" {
			product.Description = nil
		} else {
			product.Description = &trimmed
		}
	}
	if product.Category != nil {
		trimmed := strings.TrimSpace(*product.Category)
		if trimmed == "" {
			product.Category = nil
		} else {
			product.Category = &trimmed
		}
	}
	if product.Unit != nil {
		trimmed := strings.TrimSpace(*product.Unit)
		if trimmed == "" {
			product.Unit = nil
		} else {
			product.Unit = &trimmed
		}
	}
}

func actorUserIDFromContext(ctx context.Context) *int64 {
	if userID, ok := auth.UserIDFromContext(ctx); ok {
		return &userID
	}
	return nil
}
