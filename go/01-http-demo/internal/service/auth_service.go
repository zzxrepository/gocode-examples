package service

import (
	"context"
	"errors"
	"time"

	"gin-demo-v1/internal/model"
	"gin-demo-v1/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	users     *repository.UserRepository
	sessions  *repository.SessionRepository
	secret    string
	expiresIn time.Duration
}

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func NewAuthService(users *repository.UserRepository, sessions *repository.SessionRepository, secret string, expiresIn time.Duration) *AuthService {
	return &AuthService{users: users, sessions: sessions, secret: secret, expiresIn: expiresIn}
}

func (s *AuthService) Register(ctx context.Context, req model.RegisterRequest) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return s.users.Create(ctx, req.Username, string(hash))
}

func (s *AuthService) Login(ctx context.Context, req model.LoginRequest) (*model.LoginResponse, error) {
	user, err := s.users.FindByUsername(ctx, req.Username)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		return nil, errors.New("invalid username or password")
	}
	token, err := s.issueToken(user)
	if err != nil {
		return nil, err
	}
	if err := s.sessions.SaveToken(ctx, user.ID, token, s.expiresIn); err != nil {
		return nil, err
	}
	return &model.LoginResponse{Token: token}, nil
}

func (s *AuthService) ParseToken(ctx context.Context, tokenValue string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenValue, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	exists, err := s.sessions.TokenExists(ctx, claims.UserID, tokenValue)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("session expired")
	}
	return claims, nil
}

func (s *AuthService) issueToken(user *model.User) (string, error) {
	now := time.Now()
	claims := Claims{UserID: user.ID, Username: user.Username, RegisteredClaims: jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(now.Add(s.expiresIn)),
		IssuedAt:  jwt.NewNumericDate(now),
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.secret))
}
