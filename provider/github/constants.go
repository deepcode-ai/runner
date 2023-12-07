package github

import (
	"errors"
	"fmt"
)

const (
	HeaderGithubSignature = "x-hub-signature-256"
	HeaderRunnerSignature = "x-deepcode-signature-256"
	HeaderRunnerID        = "x-deepcode-runner-id"
	HeaderAppID           = "x-deepcode-app-id"
	HeaderInstallationID  = "X-Installation-Id"

	HeaderContentType    = "Content-Type"
	HeaderAuthorization  = "Authorization"
	HeaderAccept         = "Accept"
	HeaderAcceptEncoding = "Accept-Encoding"

	HeaderValueGithubAccept = "application/vnd.github+json"
)

var (
	ErrInvalidSignature     = errors.New("invalid signature")
	ErrMandatoryArgsMissing = errors.New("mandatory args missing")
	ErrAppNotFound          = fmt.Errorf("app not found")
)
