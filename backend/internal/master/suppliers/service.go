package suppliers

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
	ErrSupplierNotFound      = errors.New("supplier not found")
	ErrSupplierDeleted       = errors.New("supplier deleted")
	ErrSupplierDuplicateCode = errors.New("supplier code already exists")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Supplier, error) {
	supplier, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSupplierNotFound
		}
		return nil, err
	}
	return supplier, nil
}

func (s *Service) List(ctx context.Context, filter SupplierFilter) ([]Supplier, error) {
	return s.repo.List(ctx, filter)
}

func (s *Service) Create(ctx context.Context, supplier *Supplier) (int64, error) {
	if err := validateSupplier(supplier); err != nil {
		return 0, err
	}
	normalizeOptionalStrings(supplier)

	if existing, err := s.repo.GetByCode(ctx, supplier.Code); err == nil && existing != nil {
		return 0, ErrSupplierDuplicateCode
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
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

	id, err := s.repo.CreateWithTx(ctx, tx, supplier)
	if err != nil {
		return 0, err
	}

	if s.auditSvc != nil {
		actorUserID := actorUserIDFromContext(ctx)
		resourceID := fmt.Sprintf("%d", id)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserID,
			Action:      "supplier.create",
			Resource:    "supplier",
			ResourceID:  &resourceID,
			Metadata: map[string]any{
				"code": supplier.Code,
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

func (s *Service) Update(ctx context.Context, supplier *Supplier) error {
	existing, err := s.repo.GetByIDIncludeDeleted(ctx, supplier.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSupplierNotFound
		}
		return err
	}
	if existing.DeletedAt != nil {
		return ErrSupplierDeleted
	}

	if err := validateSupplier(supplier); err != nil {
		return err
	}
	normalizeOptionalStrings(supplier)

	if existingCode, err := s.repo.GetByCode(ctx, supplier.Code); err == nil && existingCode != nil && existingCode.ID != supplier.ID {
		return ErrSupplierDuplicateCode
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
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

	if err := s.repo.UpdateWithTx(ctx, tx, supplier); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSupplierNotFound
		}
		return err
	}

	if s.auditSvc != nil {
		actorUserID := actorUserIDFromContext(ctx)
		resourceID := fmt.Sprintf("%d", supplier.ID)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserID,
			Action:      "supplier.update",
			Resource:    "supplier",
			ResourceID:  &resourceID,
			Metadata: map[string]any{
				"code": supplier.Code,
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
			return ErrSupplierNotFound
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
			return ErrSupplierNotFound
		}
		return err
	}

	if s.auditSvc != nil {
		actorUserID := actorUserIDFromContext(ctx)
		resourceID := fmt.Sprintf("%d", id)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserID,
			Action:      "supplier.delete",
			Resource:    "supplier",
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

func validateSupplier(supplier *Supplier) error {
	if strings.TrimSpace(supplier.Code) == "" {
		return &ValidationError{Field: "code", Message: "code is required"}
	}
	if strings.TrimSpace(supplier.Name) == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	return nil
}

func normalizeOptionalStrings(supplier *Supplier) {
	if supplier.Phone != nil {
		trimmed := strings.TrimSpace(*supplier.Phone)
		if trimmed == "" {
			supplier.Phone = nil
		} else {
			supplier.Phone = &trimmed
		}
	}
	if supplier.Email != nil {
		trimmed := strings.TrimSpace(*supplier.Email)
		if trimmed == "" {
			supplier.Email = nil
		} else {
			supplier.Email = &trimmed
		}
	}
	if supplier.Address != nil {
		trimmed := strings.TrimSpace(*supplier.Address)
		if trimmed == "" {
			supplier.Address = nil
		} else {
			supplier.Address = &trimmed
		}
	}
	supplier.Code = strings.TrimSpace(supplier.Code)
	supplier.Name = strings.TrimSpace(supplier.Name)
}

func actorUserIDFromContext(ctx context.Context) *int64 {
	if userID, ok := auth.UserIDFromContext(ctx); ok {
		return &userID
	}
	return nil
}
