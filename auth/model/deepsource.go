package model

import (
	"crypto/rsa"
	"net/url"
)

type DeepCode struct {
	Host      url.URL
	PublicKey *rsa.PublicKey
}
