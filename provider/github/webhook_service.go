package github

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/deepcode-ai/runner/forwarder"
	"github.com/deepcode-ai/runner/httperror"
	"github.com/deepcode-ai/runner/provider/model"
	"golang.org/x/exp/slog"
)

type WebhookService struct {
	appFactory *AppFactory
	runner     *model.Runner
	deepcode *model.DeepCode
	client     *http.Client
}

func NewWebhookService(appFactory *AppFactory, runner *model.Runner, deepcode *model.DeepCode, client *http.Client) *WebhookService {
	return &WebhookService{
		appFactory: appFactory,
		runner:     runner,
		deepcode: deepcode,
		client:     client,
	}
}

// Process processes the webhook request.  It verifies the signature, adds a
// signature for the cloud server, and then proxies the request to the cloud.
func (s *WebhookService) Process(req *WebhookRequest) (*http.Response, error) {
	app := s.appFactory.GetApp(req.AppID)
	if app == nil {
		return nil, httperror.ErrAppInvalid(nil)
	}

	// Read body and rewind.  We need the body to process signatures.  The body
	// is also needed to proxy the request to the cloud server.
	body, err := io.ReadAll(req.HTTPRequest.Body)
	if err != nil {
		return nil, httperror.ErrUnknown(
			fmt.Errorf("failed to read request body: %w", err),
		)
	}
	req.HTTPRequest.Body = io.NopCloser(bytes.NewReader(body)) // rewind body

	if err := app.VerifyWebhookSignature(body, req.Signature); err != nil {
		return nil, httperror.ErrUnauthorized(
			fmt.Errorf("failed to verify webhook signature: %w", err),
		)
	}

	// generate signature for cloud server
	signature, err := s.runner.SignPayload(body)
	if err != nil {
		err = fmt.Errorf("failed to sign payload: %w", err)
		return nil, httperror.ErrUnknown(err)
	}

	header := s.prepareHeader(app, signature)

	f := forwarder.New(s.client)

	res, err := f.Forward(req.HTTPRequest, &forwarder.Opts{
		TargetURL: *s.deepcode.WebhookURL(),
		Headers:   header,
		Query:     nil,
	})

	slog.Info("Status code from DeepCode", slog.Int("status_code", res.StatusCode))

	if err != nil {
		err := fmt.Errorf("failed to proxy request: %w", err)
		return nil, httperror.ErrUpstreamFailed(err)
	}

	return res, nil
}

func (s *WebhookService) prepareHeader(app *App, signature string) http.Header {
	header := http.Header{}
	header.Set(HeaderRunnerID, s.runner.ID)
	header.Set(HeaderAppID, app.ID)
	header.Set(HeaderRunnerSignature, signature)
	header.Set(HeaderContentType, "application/json")
	return header
}
