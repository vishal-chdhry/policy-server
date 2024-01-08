package storage

import (
	"github.com/k3s-io/kine/pkg/client"
	"github.com/kyverno/policy-server/pkg/storage/inmemory"
	"github.com/kyverno/policy-server/pkg/storage/kine"
	"k8s.io/klog/v2"
)

type Storage interface {
	client.Client
}

func NewStorage(debug bool) (Storage, error) {
	klog.Info("setting up storage", "debug=", debug)
	if debug {
		return inmemory.New(), nil
	}
	kineClient, err := kine.New()
	if err != nil {
		return nil, err
	}
	return kineClient, nil
}
