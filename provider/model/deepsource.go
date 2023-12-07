package model

import "net/url"

type DeepCode struct {
	Host url.URL
}

func (d *DeepCode) WebhookURL() *url.URL {
	return d.Host.JoinPath("/services/webhooks/github/")
}
