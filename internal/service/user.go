package service

import (
	"context"
	"errors"
	"strings"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) List(ctx context.Context) ([]*model.User, error) {
	return s.repo.List(ctx)
}

func (s *UserService) GetByID(ctx context.Context, id int64) (*model.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) Create(ctx context.Context, username, password, role string) (*model.User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, errors.New("el nombre de usuario es requerido")
	}
	if len(password) < 4 {
		return nil, errors.New("la contraseña debe tener al menos 4 caracteres")
	}
	if role != "admin" && role != "cashier" {
		return nil, errors.New("rol inválido: debe ser admin o cashier")
	}

	// Check username uniqueness
	if _, err := s.repo.GetByUsername(ctx, username); err == nil {
		return nil, errors.New("el nombre de usuario ya existe")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("error procesando contraseña")
	}

	return s.repo.Create(ctx, &model.User{
		Username:     username,
		PasswordHash: string(hash),
		Role:         role,
	})
}

func (s *UserService) Update(ctx context.Context, id int64, username, role string, active bool, password string) (*model.User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, errors.New("el nombre de usuario es requerido")
	}
	if role != "admin" && role != "cashier" {
		return nil, errors.New("rol inválido: debe ser admin o cashier")
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.New("usuario no encontrado")
	}

	// Prevent removing last admin
	if (existing.Role == "admin" && role != "admin") || (existing.Active && !active && existing.Role == "admin") {
		admins, err := s.repo.CountAdmins(ctx)
		if err != nil {
			return nil, err
		}
		if admins <= 1 {
			return nil, errors.New("no se puede modificar el único administrador activo")
		}
	}

	// Check username uniqueness if changed
	if existing.Username != username {
		if _, err := s.repo.GetByUsername(ctx, username); err == nil {
			return nil, errors.New("el nombre de usuario ya existe")
		}
	}

	if _, err := s.repo.Update(ctx, &model.User{
		ID:       id,
		Username: username,
		Role:     role,
		Active:   active,
	}); err != nil {
		return nil, err
	}

	// Update password only if provided
	if pw := strings.TrimSpace(password); pw != "" {
		if len(pw) < 4 {
			return nil, errors.New("la contraseña debe tener al menos 4 caracteres")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
		if err != nil {
			return nil, errors.New("error procesando contraseña")
		}
		if err := s.repo.UpdatePassword(ctx, id, string(hash)); err != nil {
			return nil, err
		}
	}

	return s.repo.GetByID(ctx, id)
}

func (s *UserService) ChangePassword(ctx context.Context, id int64, currentPassword, newPassword string) error {
	newPassword = strings.TrimSpace(newPassword)
	if len(newPassword) < 6 {
		return errors.New("la nueva contraseña debe tener al menos 6 caracteres")
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return errors.New("usuario no encontrado")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return errors.New("contraseña actual incorrecta")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("error procesando contraseña")
	}

	return s.repo.UpdatePassword(ctx, id, string(hash))
}

func (s *UserService) Deactivate(ctx context.Context, id int64) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return errors.New("usuario no encontrado")
	}
	if existing.Role == "admin" {
		admins, err := s.repo.CountAdmins(ctx)
		if err != nil {
			return err
		}
		if admins <= 1 {
			return errors.New("no se puede desactivar el único administrador activo")
		}
	}
	return s.repo.Deactivate(ctx, id)
}
