package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/order-processing-system/order-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type mockAuthRepo struct {
	repository.Querier
	users map[string]repository.User
}

func (m *mockAuthRepo) WithTx(ctx context.Context, fn func(q repository.Querier) error) error {
	return fn(m)
}

func (m *mockAuthRepo) CreateUser(ctx context.Context, arg repository.CreateUserParams) (repository.User, error) {
	if m.users == nil {
		m.users = make(map[string]repository.User)
	}
	for _, u := range m.users {
		if u.Email == arg.Email {
			return repository.User{}, errors.New("email already exists")
		}
	}
	u := repository.User{
		ID:           uuid.New(),
		Email:        arg.Email,
		PasswordHash: arg.PasswordHash,
		Role:         arg.Role,
		CreatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	m.users[arg.Email] = u
	return u, nil
}

func (m *mockAuthRepo) GetUserByEmail(ctx context.Context, email string) (repository.User, error) {
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	return repository.User{}, errors.New("user not found")
}

func TestAuthService_Register(t *testing.T) {
	repo := &mockAuthRepo{}
	svc := NewAuthService(repo)

	// Valid registration
	u, err := svc.Register(context.Background(), "test@test.com", "password123", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Role != "customer" {
		t.Errorf("expected default role customer, got %s", u.Role)
	}

	// Password should be hashed
	if u.PasswordHash == "password123" {
		t.Errorf("expected hashed password, got raw text")
	}
}

func TestAuthService_Login(t *testing.T) {
	repo := &mockAuthRepo{}
	svc := NewAuthService(repo)

	// Setup user
	hash, _ := bcrypt.GenerateFromPassword([]byte("mypassword"), bcrypt.DefaultCost)
	repo.CreateUser(context.Background(), repository.CreateUserParams{
		Email:        "user@test.com",
		PasswordHash: string(hash),
		Role:         "customer",
	})

	// Valid login
	token, err := svc.Login(context.Background(), "user@test.com", "mypassword")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Errorf("expected JWT token, got empty string")
	}

	// Invalid password
	_, err = svc.Login(context.Background(), "user@test.com", "wrongpass")
	if err == nil {
		t.Fatalf("expected error for wrong password")
	}

	// Invalid email
	_, err = svc.Login(context.Background(), "wrong@test.com", "mypassword")
	if err == nil {
		t.Fatalf("expected error for non-existent user")
	}
}
