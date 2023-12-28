package api

import (
	"k8s.io/apiserver/pkg/registry/rest"
)

type API interface {
	rest.Storage
	// rest.StandardStorage // Get, List, Create, Update, Delete, DeleteCollection, Watch
	rest.KindProvider
	rest.Scoper
	rest.SingularNameProvider
}
