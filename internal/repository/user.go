package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/andreyo/pos/internal/model"
)

type userRepo struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) List(ctx context.Context) ([]*model.User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, username, role, active, created_at FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("user list: %w", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		u := &model.User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.Active, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("user scan: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, username, role, active, created_at FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Username, &u.Role, &u.Active, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, username, role, active, created_at FROM users WHERE username = ?`, username).
		Scan(&u.ID, &u.Username, &u.Role, &u.Active, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (r *userRepo) Create(ctx context.Context, u *model.User) (*model.User, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, role, active) VALUES (?, ?, ?, 1)`,
		u.Username, u.PasswordHash, u.Role)
	if err != nil {
		return nil, fmt.Errorf("user create: %w", err)
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *userRepo) Update(ctx context.Context, u *model.User) (*model.User, error) {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET username = ?, role = ?, active = ? WHERE id = ?`,
		u.Username, u.Role, u.Active, u.ID)
	if err != nil {
		return nil, fmt.Errorf("user update: %w", err)
	}
	return r.GetByID(ctx, u.ID)
}

func (r *userRepo) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, id)
	return err
}

func (r *userRepo) Deactivate(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET active = 0 WHERE id = ?`, id)
	return err
}

func (r *userRepo) CountAdmins(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users WHERE role = 'admin' AND active = 1`).Scan(&count)
	return count, err
}
