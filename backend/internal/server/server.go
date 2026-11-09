// Package server は Echo のサーバーを組み立てる。
package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	oapimiddleware "github.com/oapi-codegen/echo-middleware"

	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/oapi"
)

// MaxBodyBytes はリクエストボディの上限（バイト）。超えたら 413 を返す。
// 仕様書の検証はボディを全部メモリに読み込んでから行うため、その前に大きすぎるボディを拒否する。
//
// 仕様書の maxLength は文字数で数えるが、絵文字などの BMP 外の文字は JSON で
// サロゲートペア（\uD83D\uDE00 のような 12 バイト）にエスケープできる。
// signup で仕様書の検証を通る最大のボディは約 4.1KB（TestSignup_MaxValidBodyFitsLimit）のため、
// 余裕を持たせて 8KB にする。
const MaxBodyBytes = 8 * 1024

func init() {
	// kin-openapi は format: email を標準では検証しないため登録する。
	openapi3.DefineStringFormatValidator("email", openapi3.NewRegexpFormatValidator(openapi3.FormatOfStringForEmail))
}

// New は API のルーティングとミドルウェアを設定した Echo を返す。
// リクエストは OpenAPI の仕様書で検証してからハンドラーに渡す。
func New(h oapi.StrictServerInterface, logger *slog.Logger) (*echo.Echo, error) {
	spec, err := oapi.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("load openapi spec: %w", err)
	}
	// servers の Host 検証を行わない（デプロイ先のホスト名に依存させない）。
	spec.Servers = nil

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = errorHandler(logger)

	e.Use(middleware.Recover())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod:  true,
		LogURI:     true,
		LogStatus:  true,
		LogLatency: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.InfoContext(c.Request().Context(), "request",
				"method", v.Method, "uri", v.URI, "status", v.Status, "latency", v.Latency)
			return nil
		},
	}))
	e.Use(middleware.BodyLimitWithConfig(middleware.BodyLimitConfig{Limit: fmt.Sprintf("%dB", MaxBodyBytes)}))
	e.Use(oapimiddleware.OapiRequestValidatorWithOptions(spec, &oapimiddleware.Options{
		SilenceServersWarning: true,
	}))

	oapi.RegisterHandlers(e, oapi.NewStrictHandler(h, nil))
	return e, nil
}

// errorHandler はエラーを仕様書の Error 形式（{"message": "..."}）で返す。
// サーバー側のエラー（5xx）と想定外のエラーは内容をログにだけ出し、クライアントには詳細を返さない。
func errorHandler(logger *slog.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		code := http.StatusInternalServerError
		message := http.StatusText(code)
		var he *echo.HTTPError
		if errors.As(err, &he) && he.Code < http.StatusInternalServerError {
			code = he.Code
			message = fmt.Sprint(he.Message)
		} else {
			if he != nil {
				code = he.Code
				message = http.StatusText(code)
			}
			// err.Error() には HTTPError の Internal（元のエラー）も含まれる
			logger.ErrorContext(c.Request().Context(), "unhandled error", "error", err)
		}

		if err := c.JSON(code, oapi.Error{Message: message}); err != nil {
			logger.ErrorContext(c.Request().Context(), "write error response", "error", err)
		}
	}
}
