package user

import (
	"context"
	"errors"
	"time"

	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/user-service/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	users     *Repository
	sessions  *SessionRepository
	secret    string
	expiresIn time.Duration
}

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func NewService(users *Repository, sessions *SessionRepository, secret string, expiresIn time.Duration) *Service {
	return &Service{users: users, sessions: sessions, secret: secret, expiresIn: expiresIn}
}

func (s *Service) Register(ctx context.Context, req model.RegisterRequest) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return s.users.Create(ctx, req.Username, string(hash))
}

func (s *Service) Login(ctx context.Context, req model.LoginRequest) (*model.LoginResponse, error) {
	user, err := s.users.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, errors.New("invalid username or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
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

func (s *Service) ValidateToken(ctx context.Context, tokenValue string) (*model.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenValue, &Claims{}, func(token *jwt.Token) (interface{}, error) {
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
	return &model.TokenClaims{UserID: claims.UserID, Username: claims.Username}, nil
}

func (s *Service) issueToken(user *model.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiresIn)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secret))
}
