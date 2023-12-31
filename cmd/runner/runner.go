package main

import (
	"fmt"
	"net/http"
	"time"

	runnerconfig "github.com/deepcode-ai/runner/config"
	runnermiddleware "github.com/deepcode-ai/runner/middleware"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/exp/slog"
)

var version string

const (
	Banner = `________
___  __ \___  ____________________________
__  /_/ /  / / /_  __ \_  __ \  _ \_  ___/
_  _, _// /_/ /_  / / /  / / /  __/  /    
/_/ |_| \__,_/ /_/ /_//_/ /_/\___//_/     
------------------------------------------
By DeepCode | %s
------------------------------------------`
)

const (
	DriverKubernetes = "kubernetes"
	DriverPrinter    = "printer"
	CleanupInterval  = 5 * time.Minute
)

type Server struct {
	*echo.Echo
	*runnerconfig.Config
	*http.Client
	cors echo.MiddlewareFunc
}

func NewServer(c *runnerconfig.Config) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.Recover())
	if c.Sentry != nil && c.Sentry.DSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn: c.Sentry.DSN,
		}); err != nil {
			slog.Error("failed to initialize sentry", slog.Any("err", err))
		}
		e.HTTPErrorHandler = RunnerHTTPErrorHandler
	}
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "time=${time_rfc3339_nano} level=INFO method=${method}, uri=${uri}, status=${status}\n",
	}))
	cors := runnermiddleware.CorsMiddleware(c.DeepCode.Host.String())
	return &Server{Echo: e, Config: c, cors: cors}
}

func (s *Server) Start() error {
	err := s.Echo.Start(fmt.Sprintf(":%d", RunnerPort))
	if err != nil {
		slog.Error("failed to start server", slog.Any("err", err))
		return err
	}
	return nil
}

func (*Server) PrintBanner() {
	fmt.Println(fmt.Sprintf(Banner, version))
}

func (s *Server) Router() (*Router, error) {
	router := &Router{
		e: s.Echo,
		Routes: []Route{
			{
				Method: http.MethodGet, Path: "/readyz", HandlerFunc: func(c echo.Context) error {
					return c.NoContent(http.StatusOK)
				},
			},
			{
				Method: http.MethodOptions, Path: "/*", HandlerFunc: func(c echo.Context) error { return c.NoContent(http.StatusOK) }, Middleware: []echo.MiddlewareFunc{s.cors},
			},
		},
	}
	return router, nil
}
