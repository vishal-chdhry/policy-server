package storage

import (
	"github.com/k3s-io/kine/pkg/client"
	"github.com/kyverno/policy-server/pkg/storage/inmemory"
)

type Storage interface {
	client.Client
}

func NewStorage() Storage {
	return inmemory.New()
}
