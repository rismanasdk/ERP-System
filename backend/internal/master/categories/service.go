package categories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"erp-system/backend/internal/audit"
	"erp-system/backend/internal/auth"

	"github.com/lib/pq"
)

var (
	ErrNotFound        = errors.New("category not found")
	ErrNameRequired    = errors.New("category name is required")
	ErrDuplicate       = errors.New("category name already exists")
	ErrInUse           = errors.New("category is used by products")
	ErrVersionConflict = errors.New("category was modified by another request")
	ErrVersionRequired = errors.New("expected_version must be greater than zero")
)

type Service struct {
	repo     *Repository
	auditSvc auditRecorder
}

type auditRecorder interface {
	RecordWithTx(context.Context, *sql.Tx, audit.AuditLog) (int64, error)
}

func NewService(repo *Repository, auditSvcs ...auditRecorder) *Service {
	var auditSvc auditRecorder
	if len(auditSvcs) > 0 {
		auditSvc = auditSvcs[0]
	}
	return &Service{repo: repo, auditSvc: auditSvc}
}
func (s *Service) List(ctx context.Context, filter Filter) ([]Category, error) {
	return s.repo.List(ctx, filter)
}
func (s *Service) Get(ctx context.Context, id int64) (*Category, error) {
	item, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return item, err
}
func (s *Service) Create(ctx context.Context, item *Category) (int64, error) {
	if err := validate(item); err != nil {
		return 0, err
	}
	if existing, err := s.repo.GetByName(ctx, item.Name); err == nil && existing != nil {
		return 0, ErrDuplicate
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
	id, err := s.repo.CreateWithTx(ctx, tx, item)
	if isUniqueViolation(err) {
		return 0, ErrDuplicate
	}
	if err != nil {
		return 0, err
	}
	if err = s.recordAudit(ctx, tx, audit.AuditLog{ActorUserID: actorUserIDFromContext(ctx), Action: "category.create", Resource: "product_category", ResourceID: stringID(id), Metadata: categoryAuditMetadata(item)}); err != nil {
		return 0, err
	}
	err = tx.Commit()
	return id, err
}
func (s *Service) Update(ctx context.Context, item *Category) error {
	if item.Version <= 0 {
		return ErrVersionRequired
	}
	if err := validate(item); err != nil {
		return err
	}
	if _, err := s.Get(ctx, item.ID); err != nil {
		return err
	}
	if existing, err := s.repo.GetByName(ctx, item.Name); err == nil && existing != nil && existing.ID != item.ID {
		return ErrDuplicate
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
	err = s.repo.UpdateWithTx(ctx, tx, item)
	if errors.Is(err, sql.ErrNoRows) {
		_, stateErr := s.repo.GetVersionStateWithTx(ctx, tx, item.ID)
		if errors.Is(stateErr, sql.ErrNoRows) {
			return ErrNotFound
		}
		if stateErr != nil {
			return stateErr
		}
		err = ErrVersionConflict
		return err
	}
	if isUniqueViolation(err) {
		return ErrDuplicate
	}
	if err != nil {
		return err
	}
	if err = s.recordAudit(ctx, tx, audit.AuditLog{ActorUserID: actorUserIDFromContext(ctx), Action: "category.update", Resource: "product_category", ResourceID: stringID(item.ID), Metadata: categoryAuditMetadata(item)}); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Service) Delete(ctx context.Context, id int64) error {
	item, getErr := s.Get(ctx, id)
	if getErr != nil {
		return getErr
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
	err = s.repo.DeleteWithTx(ctx, tx, id)
	if errors.Is(err, sql.ErrNoRows) {
		if item.ProductCount > 0 {
			return ErrInUse
		}
	}
	if err != nil {
		return err
	}
	if err = s.recordAudit(ctx, tx, audit.AuditLog{ActorUserID: actorUserIDFromContext(ctx), Action: "category.delete", Resource: "product_category", ResourceID: stringID(id), Metadata: categoryAuditMetadata(item)}); err != nil {
		return err
	}
	return tx.Commit()
}
func validate(item *Category) error {
	if item == nil || strings.TrimSpace(item.Name) == "" {
		return ErrNameRequired
	}
	item.Name = strings.TrimSpace(item.Name)
	return nil
}

func isUniqueViolation(err error) bool {
	pqErr, ok := err.(*pq.Error)
	return ok && pqErr.Code == "23505"
}

func (s *Service) recordAudit(ctx context.Context, tx *sql.Tx, auditLog audit.AuditLog) error {
	if s.auditSvc == nil {
		return nil
	}
	_, err := s.auditSvc.RecordWithTx(ctx, tx, auditLog)
	return err
}

func actorUserIDFromContext(ctx context.Context) *int64 {
	if userID, ok := auth.UserIDFromContext(ctx); ok {
		return &userID
	}
	return nil
}

func stringID(id int64) *string {
	value := fmt.Sprintf("%d", id)
	return &value
}

func categoryAuditMetadata(item *Category) map[string]any {
	return map[string]any{"name": item.Name, "is_active": item.IsActive}
}
