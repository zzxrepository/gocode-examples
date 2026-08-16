package repository

import (
	"context"
	"database/sql"

	"gin-demo-v1/internal/model"
)

type UserRepository struct{ db *sql.DB }

func NewUserRepository(db *sql.DB) *UserRepository { return &UserRepository{db: db} }

func (r *UserRepository) Create(ctx context.Context, username, passwordHash string) (*model.User, error) {
	result, err := r.db.ExecContext(ctx, "INSERT INTO users (username, password_hash) VALUES (?, ?)", username, passwordHash)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *UserRepository) FindByID(ctx context.Context, id int64) (*model.User, error) {
	const query = "SELECT id, username, password_hash, created_at, updated_at FROM users WHERE id = ?"
	user := new(model.User)
	if err := r.db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	const query = "SELECT id, username, password_hash, created_at, updated_at FROM users WHERE username = ?"
	user := new(model.User)
	if err := r.db.QueryRowContext(ctx, query, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return nil, err
	}
	return user, nil
}
