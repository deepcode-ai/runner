package saml

import (
	"net/http"
	"time"

	"github.com/crewjam/saml/samlsp"
	"github.com/deepcode-ai/runner/auth/model"
	"github.com/deepcode-ai/runner/auth/store"
	"github.com/deepcode-ai/runner/auth/token"
	"github.com/labstack/echo/v4"
	"github.com/segmentio/ksuid"
	"golang.org/x/exp/slog"
	"golang.org/x/oauth2"
)

type Handler struct {
	runner       *model.Runner
	deepcode   *model.DeepCode
	middleware   *samlsp.Middleware
	tokenService *token.Service
	store        store.Store
}

func NewHandler(runner *model.Runner, deepcode *model.DeepCode, middleware *samlsp.Middleware, tokenService *token.Service, store store.Store) *Handler {
	return &Handler{
		runner:       runner,
		deepcode:   deepcode,
		middleware:   middleware,
		tokenService: tokenService,
		store:        store,
	}
}

type AuthorizationRequest struct {
	ClientID string
	Scopes   string
	State    string
}

func (r *AuthorizationRequest) Parse(req *http.Request) {
	q := req.URL.Query()
	r.ClientID = q.Get("client_id")
	r.Scopes = q.Get("scopes")
	r.State = q.Get("state")
}

func (h *Handler) SAMLHandler() echo.HandlerFunc {
	return echo.WrapHandler(h.middleware)
}

func (h *Handler) AuthorizationHandler() echo.HandlerFunc {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request := new(AuthorizationRequest)
		request.Parse(r)
		if !h.runner.IsValidClientID(request.ClientID) {
			w.WriteHeader(http.StatusBadRequest)
			if _, err := w.Write([]byte("invalid client_id")); err != nil {
				slog.Error("error writing response", slog.Any("err", err))
				return
			}
			return
		}

		s, err := h.middleware.Session.GetSession(r)
		if err == samlsp.ErrNoSession {
			h.middleware.HandleStartAuthFlow(w, r)
			return
		}
		if err != nil {
			h.middleware.OnError(w, r, err)
			return
		}

		session, ok := s.(samlsp.SessionWithAttributes)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			if _, err := w.Write([]byte("unauthorized")); err != nil {
				slog.Error("error writing response", slog.Any("err", err))
				return
			}

			return
		}
		attr := session.GetAttributes()

		user := &model.User{
			Login: attr.Get("login"),
			Email: attr.Get("email"),
			Name:  attr.Get("first_name") + " " + attr.Get("last_name"),
		}
		accessToken, err := h.tokenService.GenerateToken(h.runner.ID, []string{token.ScopeUser, token.ScopeCodeRead}, user, token.ExpiryAccessToken)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			if _, err := w.Write([]byte(err.Error())); err != nil {
				slog.Error("error writing response", slog.Any("err", err))
				return
			}
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    accessToken,
			Path:     "/",
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
			HttpOnly: true,
		})

		refreshToken, err := h.tokenService.GenerateToken(h.runner.ID, []string{token.ScopeRefresh}, user, token.ExpiryRefreshToken)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			if _, err := w.Write([]byte(err.Error())); err != nil {
				slog.Error("error writing response", slog.Any("err", err))
				return
			}
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh",
			Value:    refreshToken,
			Path:     "/refresh",
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
			HttpOnly: true,
		})

		http.Redirect(w, r, "/apps/saml/auth/session?state="+request.State, http.StatusTemporaryRedirect)
	})

	return echo.WrapHandler(handler)
}

type SessionRequest struct {
	State string `query:"state"`
}

func (h *Handler) HandleSession(c echo.Context) error {
	req := new(SessionRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(400, err.Error())
	}

	cookie, err := c.Cookie("session")
	if err != nil {
		return c.JSON(400, err.Error())
	}

	user, err := h.tokenService.ReadToken(h.runner.ID, token.ScopeUser, cookie.Value)
	if err != nil {
		return c.JSON(400, err.Error())
	}

	code := ksuid.New().String()
	if err := h.store.SetAccessCode(code, user); err != nil {
		return c.JSON(400, err.Error())
	}

	u := h.deepcode.Host.JoinPath("/accounts/runner/apps/saml/login/callback/bifrost/")
	q := u.Query()
	q.Add("app_id", "saml")
	q.Add("code", code)
	q.Add("state", req.State)
	u.RawQuery = q.Encode()

	return c.Redirect(http.StatusTemporaryRedirect, u.String())
}

type TokenRequest struct {
	Code         string `query:"code" json:"code"`
	ClientID     string `query:"client_id" json:"client_id"`
	ClientSecret string `query:"client_secret" json:"client_secret"`
}

func (h *Handler) HandleToken(c echo.Context) error {
	req := new(TokenRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(400, err.Error())
	}
	if !h.runner.IsValidClientID(req.ClientID) || !h.runner.IsValidClientSecret(req.ClientSecret) {
		return c.JSON(400, "invalid client_id or client_secret")
	}

	user, err := h.store.VerifyAccessCode(req.Code)
	if err != nil {
		return c.JSON(http.StatusForbidden, err.Error())
	}

	accessToken, err := h.tokenService.GenerateToken(h.runner.ID, []string{token.ScopeUser}, user, token.ExpiryAccessToken)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	refreshtoken, err := h.tokenService.GenerateToken(h.runner.ID, []string{token.ScopeRefresh}, user, token.ExpiryRefreshToken)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshtoken,
		Expiry:       time.Now().Add(24 * time.Minute),
		TokenType:    "Bearer",
	})
}

type RefreshRequest struct {
	AppID        string `param:"app_id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) HandleRefresh(c echo.Context) error {
	req := &RefreshRequest{}
	if err := c.Bind(req); err != nil {
		return c.JSON(400, err.Error())
	}

	if !(h.runner.IsValidClientID(req.ClientID) || !h.runner.IsValidClientSecret(req.ClientSecret)) {
		return c.JSON(400, "invalid client_id or client_secret")
	}

	user, err := h.tokenService.ReadToken(h.runner.ID, token.ScopeRefresh, req.RefreshToken)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, err.Error())
	}

	accessToken, err := h.tokenService.GenerateToken(h.runner.ID, []string{token.ScopeUser}, user, token.ExpiryAccessToken)
	if err != nil {
		return c.JSON(500, err.Error())
	}

	return c.JSON(200, &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: req.RefreshToken,
		Expiry:       time.Now().Add(15 * time.Minute),
		TokenType:    "Bearer",
	})
}
