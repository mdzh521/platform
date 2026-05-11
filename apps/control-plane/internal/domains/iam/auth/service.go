package auth

import (
	"errors"
	"time"

	"backend-center/internal/config"
	"backend-center/internal/domains/iam/rbac"
	"backend-center/internal/domains/iam/user"

	"github.com/golang-jwt/jwt/v5"
)

type Service struct {
	cfg         config.Config
	userService *user.Service
	rbacService *rbac.Service
}

func NewService(cfg config.Config, userService *user.Service, rbacService *rbac.Service) *Service {
	return &Service{cfg: cfg, userService: userService, rbacService: rbacService}
}

type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type Claims struct {
	UserID      uint     `json:"user_id"`
	Username    string   `json:"username"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

func (s *Service) LoginLocal(input LoginInput) (map[string]any, error) {
	model, err := s.userService.FindByUsername(input.Username)
	if err != nil {
		return nil, errors.New("invalid username or password")
	}
	if model.Status != "active" {
		return nil, errors.New("user is not active")
	}
	if model.SourceType != user.SourceLocal {
		return nil, errors.New("this account is managed by an external identity source")
	}
	if err := s.userService.VerifyLocalPassword(model.ID, input.Password); err != nil {
		return nil, errors.New("invalid username or password")
	}

	permissions, err := s.rbacService.PermissionsForUser(model.ID)
	if err != nil {
		return nil, err
	}
	_ = s.userService.TouchLogin(model.ID)

	expiresAt := time.Now().Add(12 * time.Hour)
	claims := Claims{
		UserID:      model.ID,
		Username:    model.Username,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   model.Username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"token":       signed,
		"expires_at":  expiresAt,
		"user":        model,
		"permissions": permissions,
	}, nil
}
