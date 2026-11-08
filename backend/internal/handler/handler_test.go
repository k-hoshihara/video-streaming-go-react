package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/domain"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/oapi"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/usecase"
)

type fakeUserUsecase struct {
	user *domain.User
	err  error
	in   usecase.SignupInput
}

func (f *fakeUserUsecase) Signup(_ context.Context, in usecase.SignupInput) (*domain.User, error) {
	f.in = in
	return f.user, f.err
}

func signupRequest() oapi.SignupRequestObject {
	return oapi.SignupRequestObject{Body: &oapi.SignupJSONRequestBody{
		Name: "taro", Email: "taro@example.com", Password: "password123",
	}}
}

func TestHandler_Signup_Created(t *testing.T) {
	t.Parallel()

	user := &domain.User{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "taro",
		Email:     "taro@example.com",
		CreatedAt: time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC),
	}
	uc := &fakeUserUsecase{user: user}

	res, err := New(uc).Signup(t.Context(), signupRequest())
	if err != nil {
		t.Fatalf("Signup() error = %v", err)
	}

	want := oapi.Signup201JSONResponse{Id: user.ID, Name: "taro", Email: "taro@example.com", CreatedAt: user.CreatedAt}
	if res != want {
		t.Errorf("response = %+v, want %+v", res, want)
	}
	if uc.in != (usecase.SignupInput{Name: "taro", Email: "taro@example.com", Password: "password123"}) {
		t.Errorf("usecase input = %+v", uc.in)
	}
}

func TestHandler_Signup_Errors(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")
	tests := []struct {
		name    string
		ucErr   error
		want    oapi.SignupResponseObject
		wantErr error
	}{
		{"password too long", domain.ErrPasswordTooLong, oapi.Signup400JSONResponse{Message: "password is too long"}, nil},
		{"email already exists", domain.ErrEmailAlreadyExists, oapi.Signup409JSONResponse{Message: "email already exists"}, nil},
		// 想定外のエラーはそのまま返し、server の errorHandler で 500 にする
		{"unexpected error", errBoom, nil, errBoom},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res, err := New(&fakeUserUsecase{err: tt.ucErr}).Signup(t.Context(), signupRequest())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Signup() error = %v, want %v", err, tt.wantErr)
			}
			if res != tt.want {
				t.Errorf("response = %#v, want %#v", res, tt.want)
			}
		})
	}
}
