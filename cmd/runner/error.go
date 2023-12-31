package main

import (
	"net/http"

	"github.com/deepcode-ai/runner/httperror"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
)

func RunnerHTTPErrorHandler(err error, c echo.Context) {
	switch typedErr := err.(type) {
	case *httperror.Error:
		sentry.CaptureException(err)
		_ = c.JSON(typedErr.Code, typedErr)
	case *echo.HTTPError:
		if typedErr.Code < 500 {
			_ = c.JSON(typedErr.Code, typedErr.Message)
			return
		}
		sentry.CaptureException(err)
		_ = c.JSON(typedErr.Code, typedErr)
	default:
		sentry.CaptureException(err)
		_ = c.JSON(http.StatusInternalServerError, httperror.ErrUnknown(err))
	}
}
