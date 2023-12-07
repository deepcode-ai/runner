package main

import (
	"context"
	"net/http"

	"github.com/deepcode-ai/runner/auth/jwtutil"
	"github.com/deepcode-ai/runner/config"
	"github.com/deepcode-ai/runner/sync"
)

var providers = map[string]string{
	"github": "gh",
}

func GetSyncer(_ context.Context, c *config.Config, client *http.Client) *sync.Syncer {
	deepcode := &sync.DeepCode{
		Host: c.DeepCode.Host,
	}
	runner := &sync.Runner{
		ID:            c.Runner.ID,
		Host:          c.Runner.Host,
		ClientID:      c.Runner.ClientID,
		ClientSecret:  c.Runner.ClientSecret,
		WebhookSecret: c.Runner.WebhookSecret,
	}

	apps := make([]sync.App, 0, len(c.Apps))
	for _, a := range c.Apps {
		apps = append(apps, sync.App{
			ID:       a.ID,
			Name:     a.Name,
			Provider: providers[a.Provider],
		})
	}

	signer := jwtutil.NewSigner(c.Runner.PrivateKey)
	return sync.New(deepcode, runner, apps, signer, client)
}
