package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *Repository) Create(ctx context.Context, auditLog AuditLog) (int64, error) {
	return r.CreateWithTx(ctx, nil, auditLog)
}

func (r *Repository) CreateWithTx(ctx context.Context, tx *sql.Tx, auditLog AuditLog) (int64, error) {
	metadataJSON, err := json.Marshal(auditLog.Metadata)
	if err != nil {
		return 0, err
	}

	var id int64
	query := `
        INSERT INTO audit_logs (actor_user_id, action, resource, resource_id, metadata)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `

	exec := func(q string, args ...any) *sql.Row {
		if tx != nil {
			return tx.QueryRowContext(ctx, q, args...)
		}
		return r.db.QueryRowContext(ctx, q, args...)
	}

	err = exec(query, auditLog.ActorUserID, auditLog.Action, auditLog.Resource, auditLog.ResourceID, metadataJSON).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) List(ctx context.Context, filter AuditLogFilter) ([]AuditLog, error) {
	query := `
        SELECT id, actor_user_id, action, resource, resource_id, metadata, created_at
        FROM audit_logs
    `
	clauses := []string{}
	args := []any{}
	idx := 1

	if filter.ActorUserID != nil {
		clauses = append(clauses, `actor_user_id = $`+itoa(idx))
		args = append(args, *filter.ActorUserID)
		idx++
	}
	if filter.Action != nil {
		clauses = append(clauses, `action = $`+itoa(idx))
		args = append(args, *filter.Action)
		idx++
	}
	if filter.Resource != nil {
		clauses = append(clauses, `resource = $`+itoa(idx))
		args = append(args, *filter.Resource)
		idx++
	}
	if filter.ResourceID != nil {
		clauses = append(clauses, `resource_id = $`+itoa(idx))
		args = append(args, *filter.ResourceID)
		idx++
	}
	if filter.CreatedAfter != nil {
		clauses = append(clauses, `created_at >= $`+itoa(idx))
		args = append(args, *filter.CreatedAfter)
		idx++
	}
	if filter.CreatedBefore != nil {
		clauses = append(clauses, `created_at <= $`+itoa(idx))
		args = append(args, *filter.CreatedBefore)
		idx++
	}

	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	auditLogs := []AuditLog{}
	for rows.Next() {
		var auditLog AuditLog
		var actorUserID sql.NullInt64
		var resourceID sql.NullString
		var metadataBytes []byte
		if err := rows.Scan(
			&auditLog.ID,
			&actorUserID,
			&auditLog.Action,
			&auditLog.Resource,
			&resourceID,
			&metadataBytes,
			&auditLog.CreatedAt,
		); err != nil {
			return nil, err
		}
		if actorUserID.Valid {
			auditLog.ActorUserID = &actorUserID.Int64
		}
		if resourceID.Valid {
			auditLog.ResourceID = &resourceID.String
		}
		if len(metadataBytes) > 0 {
			if err := json.Unmarshal(metadataBytes, &auditLog.Metadata); err != nil {
				return nil, err
			}
		}
		auditLogs = append(auditLogs, auditLog)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return auditLogs, nil
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
