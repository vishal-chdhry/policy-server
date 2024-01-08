package v1beta1

import (
	"k8s.io/apiserver/pkg/registry/rest"
)

type API interface {
	rest.Storage
	rest.KindProvider
	rest.Scoper
	rest.SingularNameProvider
	rest.StandardStorage
	rest.ShortNamesProvider
}
