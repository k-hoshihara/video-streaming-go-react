// Package handler は oapi-codegen が生成した StrictServerInterface を実装し、
// HTTP の入出力（生成された型）と usecase の入出力を変換する。
package handler

import (
	"context"
	"errors"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/domain"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/oapi"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/usecase"
)

// UserUsecase は handler が必要とするユーザー操作。
type UserUsecase interface {
	Signup(ctx context.Context, in usecase.SignupInput) (*domain.User, error)
}

// AuthUsecase は handler が必要とする認証の操作。
type AuthUsecase interface {
	Login(ctx context.Context, in usecase.LoginInput) (*usecase.LoginOutput, error)
}

// Handler は API のハンドラー。
type Handler struct {
	user UserUsecase
	auth AuthUsecase
}

var _ oapi.StrictServerInterface = (*Handler)(nil)

// New は Handler を生成する。
func New(user UserUsecase, auth AuthUsecase) *Handler {
	return &Handler{user: user, auth: auth}
}

// Signup は POST /v1/signup のハンドラー。
func (h *Handler) Signup(ctx context.Context, req oapi.SignupRequestObject) (oapi.SignupResponseObject, error) {
	user, err := h.user.Signup(ctx, usecase.SignupInput{
		Name:     req.Body.Name,
		Email:    string(req.Body.Email),
		Password: req.Body.Password,
	})
	switch {
	case errors.Is(err, domain.ErrPasswordTooLong):
		return oapi.Signup400JSONResponse{Message: "password is too long"}, nil
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		return oapi.Signup409JSONResponse{Message: "email already exists"}, nil
	case err != nil:
		return nil, err
	}

	return oapi.Signup201JSONResponse{
		Id:        user.ID,
		Name:      user.Name,
		Email:     openapi_types.Email(user.Email),
		CreatedAt: user.CreatedAt,
	}, nil
}

// Login は POST /v1/login のハンドラー。
func (h *Handler) Login(ctx context.Context, req oapi.LoginRequestObject) (oapi.LoginResponseObject, error) {
	out, err := h.auth.Login(ctx, usecase.LoginInput{
		Email:    string(req.Body.Email),
		Password: req.Body.Password,
	})
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		return oapi.Login401JSONResponse{Message: "invalid email or password"}, nil
	case err != nil:
		return nil, err
	}

	return oapi.Login200JSONResponse{
		AccessToken: out.AccessToken,
		TokenType:   oapi.Bearer,
		ExpiresIn:   int(out.ExpiresIn / time.Second),
	}, nil
}
