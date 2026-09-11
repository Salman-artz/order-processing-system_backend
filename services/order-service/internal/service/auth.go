package service

import (
	"context"
	"fmt"
	"os"

	"github.com/order-processing-system/pkg/jwtutil"
	"github.com/order-processing-system/order-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication logic.
type AuthService interface {
	Register(ctx context.Context, email, password, role string) (repository.User, error)
	Login(ctx context.Context, email, password string) (string, error)
}

type authService struct {
	repo repository.Repository
}

// NewAuthService creates a new AuthService.
func NewAuthService(repo repository.Repository) AuthService {
	return &authService{repo: repo}
}

func (s *authService) Register(ctx context.Context, email, password, role string) (repository.User, error) {
	if role == "" {
		role = "customer"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return repository.User{}, fmt.Errorf("failed to hash password: %w", err)
	}

	var user repository.User
	err = s.repo.WithTx(ctx, func(q repository.Querier) error {
		u, err := q.CreateUser(ctx, repository.CreateUserParams{
			Email:        email,
			PasswordHash: string(hash),
			Role:         role,
		})
		if err != nil {
			return err
		}
		user = u
		return nil
	})

	return user, err
}

func (s *authService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "fallback-secret-for-dev"
	}
	cfg := jwtutil.Config{Secret: secret}

	token, err := jwtutil.GenerateToken(cfg, user.ID.String(), user.Email, jwtutil.Role(user.Role))
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return token, nil
}
