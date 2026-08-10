package audit

import (
	"context"
	"database/sql"
)

type AuditRepository interface {
	Create(ctx context.Context, auditLog AuditLog) (int64, error)
	CreateWithTx(ctx context.Context, tx *sql.Tx, auditLog AuditLog) (int64, error)
}

type Service struct {
	repo AuditRepository
}

func NewService(repo AuditRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Record(ctx context.Context, auditLog AuditLog) (int64, error) {
	return s.repo.Create(ctx, auditLog)
}

func (s *Service) RecordWithTx(ctx context.Context, tx *sql.Tx, auditLog AuditLog) (int64, error) {
	return s.repo.CreateWithTx(ctx, tx, auditLog)
}
