package customers

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
	ErrCustomerNotFound      = errors.New("customer not found")
	ErrCustomerDeleted       = errors.New("customer deleted")
	ErrCustomerDuplicateCode = errors.New("customer code already exists")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Customer, error) {
	customer, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}
	return customer, nil
}

func (s *Service) List(ctx context.Context, filter CustomerFilter) ([]Customer, error) {
	return s.repo.List(ctx, filter)
}

func (s *Service) Create(ctx context.Context, customer *Customer) (int64, error) {
	if err := validateCustomer(customer); err != nil {
		return 0, err
	}
	normalizeOptionalStrings(customer)

	if existing, err := s.repo.GetByCode(ctx, customer.Code); err == nil && existing != nil {
		return 0, ErrCustomerDuplicateCode
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

	id, err := s.repo.CreateWithTx(ctx, tx, customer)
	if err != nil {
		return 0, err
	}

	if s.auditSvc != nil {
		actorUserID := actorUserIDFromContext(ctx)
		resourceID := fmt.Sprintf("%d", id)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserID,
			Action:      "customer.create",
			Resource:    "customer",
			ResourceID:  &resourceID,
			Metadata: map[string]any{
				"code": customer.Code,
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

func (s *Service) Update(ctx context.Context, customer *Customer) error {
	existing, err := s.repo.GetByIDIncludeDeleted(ctx, customer.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCustomerNotFound
		}
		return err
	}
	if existing.DeletedAt != nil {
		return ErrCustomerDeleted
	}

	if err := validateCustomer(customer); err != nil {
		return err
	}
	normalizeOptionalStrings(customer)

	if existingCode, err := s.repo.GetByCode(ctx, customer.Code); err == nil && existingCode != nil && existingCode.ID != customer.ID {
		return ErrCustomerDuplicateCode
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

	if err := s.repo.UpdateWithTx(ctx, tx, customer); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCustomerNotFound
		}
		return err
	}

	if s.auditSvc != nil {
		actorUserID := actorUserIDFromContext(ctx)
		resourceID := fmt.Sprintf("%d", customer.ID)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserID,
			Action:      "customer.update",
			Resource:    "customer",
			ResourceID:  &resourceID,
			Metadata: map[string]any{
				"code": customer.Code,
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
			return ErrCustomerNotFound
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
			return ErrCustomerNotFound
		}
		return err
	}

	if s.auditSvc != nil {
		actorUserID := actorUserIDFromContext(ctx)
		resourceID := fmt.Sprintf("%d", id)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserID,
			Action:      "customer.delete",
			Resource:    "customer",
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

func validateCustomer(customer *Customer) error {
	if strings.TrimSpace(customer.Code) == "" {
		return &ValidationError{Field: "code", Message: "code is required"}
	}
	if strings.TrimSpace(customer.Name) == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	return nil
}

func normalizeOptionalStrings(customer *Customer) {
	if customer.Phone != nil {
		trimmed := strings.TrimSpace(*customer.Phone)
		if trimmed == "" {
			customer.Phone = nil
		} else {
			customer.Phone = &trimmed
		}
	}
	if customer.Email != nil {
		trimmed := strings.TrimSpace(*customer.Email)
		if trimmed == "" {
			customer.Email = nil
		} else {
			customer.Email = &trimmed
		}
	}
	if customer.Address != nil {
		trimmed := strings.TrimSpace(*customer.Address)
		if trimmed == "" {
			customer.Address = nil
		} else {
			customer.Address = &trimmed
		}
	}
	if customer.TaxID != nil {
		trimmed := strings.TrimSpace(*customer.TaxID)
		if trimmed == "" {
			customer.TaxID = nil
		} else {
			customer.TaxID = &trimmed
		}
	}
	customer.Code = strings.TrimSpace(customer.Code)
	customer.Name = strings.TrimSpace(customer.Name)
}

func actorUserIDFromContext(ctx context.Context) *int64 {
	if userID, ok := auth.UserIDFromContext(ctx); ok {
		return &userID
	}
	return nil
}
