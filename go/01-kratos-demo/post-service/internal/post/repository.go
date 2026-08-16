package post

import (
	"context"
	"database/sql"

	"gin-demo/post-service/internal/model"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, post *model.Post) (*model.Post, error) {
	result, err := r.db.ExecContext(ctx, "INSERT INTO posts (title, content, user_id) VALUES (?, ?, ?)", post.Title, post.Content, post.UserID)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *Repository) List(ctx context.Context) ([]model.Post, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, title, content, user_id, created_at, updated_at FROM posts ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]model.Post, 0)
	for rows.Next() {
		var post model.Post
		if err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.UserID, &post.CreatedAt, &post.UpdatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

func (r *Repository) FindByID(ctx context.Context, id int64) (*model.Post, error) {
	const query = "SELECT id, title, content, user_id, created_at, updated_at FROM posts WHERE id = ?"
	post := new(model.Post)
	if err := r.db.QueryRowContext(ctx, query, id).Scan(&post.ID, &post.Title, &post.Content, &post.UserID, &post.CreatedAt, &post.UpdatedAt); err != nil {
		return nil, err
	}
	return post, nil
}

func (r *Repository) Update(ctx context.Context, post *model.Post) (*model.Post, error) {
	result, err := r.db.ExecContext(ctx, "UPDATE posts SET title = ?, content = ? WHERE id = ? AND user_id = ?", post.Title, post.Content, post.ID, post.UserID)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, sql.ErrNoRows
	}
	return r.FindByID(ctx, post.ID)
}

func (r *Repository) Delete(ctx context.Context, id, userID int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM posts WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
