package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/deepcode-ai/runner/config"
	"github.com/deepcode-ai/runner/provider"
	"github.com/deepcode-ai/runner/provider/github"
	"github.com/deepcode-ai/runner/provider/model"
)

func GetProvider(_ context.Context, c *config.Config, client *http.Client) (*provider.Facade, error) {
	githubApps := createGithubApps(c)
	providerApps := createProviderApps(c)

	runner := &model.Runner{
		ID:            c.Runner.ID,
		WebhookSecret: c.Runner.WebhookSecret,
	}

	deepcode := &model.DeepCode{
		Host: c.DeepCode.Host,
	}

	appFactory := github.NewAppFactory(githubApps)

	webhookService := github.NewWebhookService(appFactory, runner, deepcode, client)
	apiService := github.NewAPIService(appFactory, client)

	githubProvider, err := github.NewHandler(webhookService, apiService, appFactory, runner, deepcode, client)
	if err != nil {
		return nil, fmt.Errorf("error initializing provider: %w", err)
	}

	return provider.NewFacade(providerApps, githubProvider), nil
}

func createGithubApps(c *config.Config) map[string]*github.App {
	apps := make(map[string]*github.App)
	for _, v := range c.Apps {
		switch {
		case v.Provider == "github":
			apps[v.ID] = &github.App{
				ID:            v.ID,
				AppID:         v.Github.AppID,
				WebhookSecret: v.Github.WebhookSecret,
				BaseHost:      v.Github.Host,
				APIHost:       v.Github.APIHost,
				AppSlug:       v.Github.Slug,
				PrivateKey:    v.Github.PrivateKey,
			}
		}
	}
	return apps
}

func createProviderApps(c *config.Config) map[string]*provider.App {
	apps := make(map[string]*provider.App)
	for _, v := range c.Apps {
		switch {
		case v.Provider == "github":
			apps[v.ID] = &provider.App{
				Provider: "github",
			}
		}
	}
	return apps
}
