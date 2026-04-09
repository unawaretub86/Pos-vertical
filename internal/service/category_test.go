package service

import (
	"context"
	"errors"
	"testing"

	"github.com/andreyo/pos/internal/model"
)

type stubCategoryRepo struct {
	last      *model.Category
	deleteErr error
}

func (s *stubCategoryRepo) Create(_ context.Context, name string) (*model.Category, error) {
	s.last = &model.Category{ID: 1, Name: name}
	return s.last, nil
}
func (s *stubCategoryRepo) CreateWithDescription(_ context.Context, name, desc string) (*model.Category, error) {
	s.last = &model.Category{ID: 1, Name: name, Description: desc}
	return s.last, nil
}
func (s *stubCategoryRepo) List(_ context.Context) ([]*model.Category, error) { return nil, nil }
func (s *stubCategoryRepo) GetByID(_ context.Context, id int64) (*model.Category, error) {
	return &model.Category{ID: id, Name: "X"}, nil
}
func (s *stubCategoryRepo) Update(_ context.Context, id int64, name string) (*model.Category, error) {
	return &model.Category{ID: id, Name: name}, nil
}
func (s *stubCategoryRepo) UpdateFull(_ context.Context, id int64, name, desc string) (*model.Category, error) {
	return &model.Category{ID: id, Name: name, Description: desc}, nil
}
func (s *stubCategoryRepo) Delete(_ context.Context, _ int64) error { return s.deleteErr }

func newCategorySvc() (*CategoryService, *stubCategoryRepo) {
	r := &stubCategoryRepo{}
	return NewCategoryService(r), r
}

func TestCategoryService_Create_OK(t *testing.T) {
	svc, stub := newCategorySvc()
	cat, err := svc.Create(context.Background(), "Bebidas", "desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cat.Name != "Bebidas" {
		t.Errorf("expected 'Bebidas', got %q", cat.Name)
	}
	if stub.last == nil {
		t.Fatal("repo was not called")
	}
}

func TestCategoryService_Create_EmptyName(t *testing.T) {
	svc, _ := newCategorySvc()
	if _, err := svc.Create(context.Background(), "   ", ""); err == nil {
		t.Fatal("expected error for blank name")
	}
}

func TestCategoryService_Create_TrimsName(t *testing.T) {
	svc, stub := newCategorySvc()
	svc.Create(context.Background(), "  Bebidas  ", "") //nolint:errcheck
	if stub.last != nil && stub.last.Name != "Bebidas" {
		t.Errorf("name should be trimmed, got %q", stub.last.Name)
	}
}

func TestCategoryService_Update_EmptyName(t *testing.T) {
	svc, _ := newCategorySvc()
	if _, err := svc.Update(context.Background(), 1, "", ""); err == nil {
		t.Fatal("expected error for blank name on update")
	}
}

func TestCategoryService_Update_OK(t *testing.T) {
	svc, _ := newCategorySvc()
	cat, err := svc.Update(context.Background(), 1, "Comida", "desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cat.Name != "Comida" {
		t.Errorf("expected 'Comida', got %q", cat.Name)
	}
}

func TestCategoryService_Delete_CallsRepo(t *testing.T) {
	svc, _ := newCategorySvc()
	if err := svc.Delete(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCategoryService_GetByID_OK(t *testing.T) {
	svc, _ := newCategorySvc()
	cat, err := svc.GetByID(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cat.ID != 42 {
		t.Errorf("expected ID 42, got %d", cat.ID)
	}
}

func TestCategoryService_Delete_WithAssignedProducts_ReturnsError(t *testing.T) {
	r := &stubCategoryRepo{deleteErr: errors.New("foreign key constraint")}
	svc := NewCategoryService(r)
	if err := svc.Delete(context.Background(), 1); err == nil {
		t.Fatal("expected error when deleting category with assigned products")
	}
}
