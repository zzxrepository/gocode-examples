package svc

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/user/internal/config"
	"golang.org/x/crypto/bcrypt"
)

type ServiceContext struct {
	Config    config.Config
	DB        *sql.DB
	Redis     *redis.Client
	expiresIn time.Duration
}
type User struct {
	ID           int64
	Email        string
	PasswordHash string
}
type Claims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := sql.Open("mysql", c.MySQLDSN)
	if err != nil {
		panic(fmt.Errorf("open mysql: %w", err))
	}
	if err = db.Ping(); err != nil {
		panic(fmt.Errorf("ping mysql: %w", err))
	}
	rdb := redis.NewClient(&redis.Options{Addr: c.RedisAddr, Password: c.RedisPassword, DB: c.RedisDB})
	if err = rdb.Ping(context.Background()).Err(); err != nil {
		panic(fmt.Errorf("ping redis: %w", err))
	}
	expiresIn, err := time.ParseDuration(c.JWTExpire)
	if err != nil {
		panic(fmt.Errorf("parse JWTExpire: %w", err))
	}
	return &ServiceContext{Config: c, DB: db, Redis: rdb, expiresIn: expiresIn}
}

func (s *ServiceContext) Register(ctx context.Context, email, password string) (User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	result, err := s.DB.ExecContext(ctx, `INSERT INTO users (email, password_hash) VALUES (?, ?)`, email, string(hash))
	if err != nil {
		return User{}, err
	}
	id, err := result.LastInsertId()
	return User{ID: id, Email: email, PasswordHash: string(hash)}, err
}

func (s *ServiceContext) Login(ctx context.Context, email, password string) (string, int64, error) {
	var user User
	err := s.DB.QueryRowContext(ctx, `SELECT id, email, password_hash FROM users WHERE email = ?`, email).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return "", 0, fmt.Errorf("invalid credentials")
	}
	now := time.Now()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{UserID: user.ID, Email: user.Email, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(now.Add(s.expiresIn)), IssuedAt: jwt.NewNumericDate(now)}}).SignedString([]byte(s.Config.JWTSecret))
	if err != nil {
		return "", 0, err
	}
	if err := s.Redis.Set(ctx, `session:`+token, user.ID, s.expiresIn).Err(); err != nil {
		return "", 0, err
	}
	return token, user.ID, nil
}

func (s *ServiceContext) ValidateToken(ctx context.Context, tokenValue string) (User, error) {
	token, err := jwt.ParseWithClaims(tokenValue, &Claims{}, func(*jwt.Token) (interface{}, error) { return []byte(s.Config.JWTSecret), nil })
	if err != nil || !token.Valid {
		return User{}, fmt.Errorf("invalid token")
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return User{}, fmt.Errorf("invalid token claims")
	}
	if _, err := s.Redis.Get(ctx, `session:`+tokenValue).Result(); err != nil {
		return User{}, fmt.Errorf("session expired")
	}
	return User{ID: claims.UserID, Email: claims.Email}, nil
}
