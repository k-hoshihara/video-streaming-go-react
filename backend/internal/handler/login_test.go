package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/domain"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/oapi"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/usecase"
)

type fakeAuthUsecase struct {
	out *usecase.LoginOutput
	err error
	in  usecase.LoginInput
}

func (f *fakeAuthUsecase) Login(_ context.Context, in usecase.LoginInput) (*usecase.LoginOutput, error) {
	f.in = in
	return f.out, f.err
}

func loginRequest() oapi.LoginRequestObject {
	return oapi.LoginRequestObject{Body: &oapi.LoginJSONRequestBody{
		Email: "taro@example.com", Password: "password123",
	}}
}

func TestHandler_Login_OK(t *testing.T) {
	t.Parallel()

	uc := &fakeAuthUsecase{out: &usecase.LoginOutput{AccessToken: "token", ExpiresIn: time.Hour}}
	res, err := New(nil, uc).Login(t.Context(), loginRequest())
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	want := oapi.Login200JSONResponse{AccessToken: "token", TokenType: oapi.Bearer, ExpiresIn: 3600}
	if res != want {
		t.Errorf("Login() = %#v, want %#v", res, want)
	}
	if uc.in != (usecase.LoginInput{Email: "taro@example.com", Password: "password123"}) {
		t.Errorf("usecase input = %+v", uc.in)
	}
}

func TestHandler_Login_Errors(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")
	tests := []struct {
		name    string
		ucErr   error
		want    oapi.LoginResponseObject
		wantErr error
	}{
		{"invalid credentials", domain.ErrInvalidCredentials,
			oapi.Login401JSONResponse{Message: "invalid email or password"}, nil},
		{"wrapped invalid credentials", errors.Join(errBoom, domain.ErrInvalidCredentials),
			oapi.Login401JSONResponse{Message: "invalid email or password"}, nil},
		// 想定外のエラーはそのまま返し、server のエラーハンドラーで 500 にする
		{"unexpected error", errBoom, nil, errBoom},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res, err := New(nil, &fakeAuthUsecase{err: tt.ucErr}).Login(t.Context(), loginRequest())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Login() error = %v, want %v", err, tt.wantErr)
			}
			if res != tt.want {
				t.Errorf("Login() = %#v, want %#v", res, tt.want)
			}
		})
	}
}
