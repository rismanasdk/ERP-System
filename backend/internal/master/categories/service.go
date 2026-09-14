package categories

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

var (
	ErrNotFound     = errors.New("category not found")
	ErrNameRequired = errors.New("category name is required")
	ErrDuplicate    = errors.New("category name already exists")
	ErrInUse        = errors.New("category is used by products")
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }
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
	return s.repo.Create(ctx, item)
}
func (s *Service) Update(ctx context.Context, item *Category) error {
	if err := validate(item); err != nil {
		return err
	}
	if _, err := s.Get(ctx, item.ID); err != nil {
		return err
	}
	if existing, err := s.repo.GetByName(ctx, item.Name); err == nil && existing != nil && existing.ID != item.ID {
		return ErrDuplicate
	}
	return s.repo.Update(ctx, item)
}
func (s *Service) Delete(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		if item, getErr := s.Get(ctx, id); getErr != nil {
			return getErr
		} else if item.ProductCount > 0 {
			return ErrInUse
		}
	}
	return err
}
func validate(item *Category) error {
	if item == nil || strings.TrimSpace(item.Name) == "" {
		return ErrNameRequired
	}
	item.Name = strings.TrimSpace(item.Name)
	return nil
}
