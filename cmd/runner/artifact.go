package main

import (
	"context"
	"fmt"

	"github.com/deepcode-ai/artifacts/storage"
	"github.com/deepcode-ai/runner/artifact"
	"github.com/deepcode-ai/runner/config"
)

func GetArtifacts(ctx context.Context, c *config.Config) (*artifact.Facade, error) {
	storage, err := storage.NewStorageClient(ctx, c.ObjectStorage.Provider, []byte(c.ObjectStorage.Credential))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize artifacts: %w", err)
	}

	opts := &artifact.Opts{
		Storage:       storage,
		Bucket:        c.ObjectStorage.Bucket,
		AllowedOrigin: c.DeepSource.Host.String(),
	}

	return artifact.New(ctx, opts)
}
