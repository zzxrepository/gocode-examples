package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionRepository struct{ rdb *redis.Client }

func NewSessionRepository(rdb *redis.Client) *SessionRepository { return &SessionRepository{rdb: rdb} }

func (r *SessionRepository) SaveToken(ctx context.Context, userID int64, token string, ttl time.Duration) error {
	return r.rdb.Set(ctx, sessionKey(userID), token, ttl).Err()
}

func (r *SessionRepository) TokenExists(ctx context.Context, userID int64, token string) (bool, error) {
	value, err := r.rdb.Get(ctx, sessionKey(userID)).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return value == token, nil
}

func sessionKey(userID int64) string { return fmt.Sprintf("blog_demo_v1:session:%d", userID) }
