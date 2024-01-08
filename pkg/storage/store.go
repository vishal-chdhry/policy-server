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

func NewStorage(debug bool) Storage {
	klog.Info("setting up storage", "debug", debug)
	if debug {
		return inmemory.New()
	}
	kineClient, _ := kine.New()
	return kineClient
}
