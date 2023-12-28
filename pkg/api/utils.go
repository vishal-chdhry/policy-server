package api

import (
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	openapicommon "k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

var (
	defNamer    = openapinamer.NewDefinitionNamer(Scheme)
	refCallback = func(name string) spec.Ref {
		defName, _ := defNamer.GetDefinitionName(name)
		return spec.MustCreateRef("#/components/schemas/" + openapicommon.EscapeJsonPointer(defName))
	}
)
