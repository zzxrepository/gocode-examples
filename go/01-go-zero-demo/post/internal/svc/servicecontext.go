package svc

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/internal/config"
)

type ServiceContext struct {
	Config config.Config
	DB     *sql.DB
}
type Post struct {
	ID      int64
	UserID  int64
	Title   string
	Content string
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := sql.Open("mysql", c.MySQLDSN)
	if err != nil {
		panic(fmt.Errorf("open mysql: %w", err))
	}
	if err = db.Ping(); err != nil {
		panic(fmt.Errorf("ping mysql: %w", err))
	}
	return &ServiceContext{Config: c, DB: db}
}

func (s *ServiceContext) List(ctx context.Context) ([]Post, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, user_id, title, content FROM posts ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.UserID, &post.Title, &post.Content); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

func (s *ServiceContext) Get(ctx context.Context, id int64) (Post, error) {
	var post Post
	err := s.DB.QueryRowContext(ctx, `SELECT id, user_id, title, content FROM posts WHERE id = ?`, id).Scan(&post.ID, &post.UserID, &post.Title, &post.Content)
	return post, err
}

func (s *ServiceContext) Create(ctx context.Context, userID int64, title, content string) (Post, error) {
	result, err := s.DB.ExecContext(ctx, `INSERT INTO posts (user_id, title, content) VALUES (?, ?, ?)`, userID, title, content)
	if err != nil {
		return Post{}, err
	}
	id, err := result.LastInsertId()
	return Post{ID: id, UserID: userID, Title: title, Content: content}, err
}

func (s *ServiceContext) Update(ctx context.Context, userID, id int64, title, content string) (Post, error) {
	result, err := s.DB.ExecContext(ctx, `UPDATE posts SET title = ?, content = ? WHERE id = ? AND user_id = ?`, title, content, id, userID)
	if err != nil {
		return Post{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return Post{}, sql.ErrNoRows
	}
	return Post{ID: id, UserID: userID, Title: title, Content: content}, nil
}

func (s *ServiceContext) Delete(ctx context.Context, userID, id int64) error {
	result, err := s.DB.ExecContext(ctx, `DELETE FROM posts WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
