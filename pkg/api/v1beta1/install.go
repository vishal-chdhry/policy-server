// Copyright 2023 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package v1beta1

import (
	"github.com/kyverno/policy-server/pkg/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"

	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1beta1"
)

// Build constructs APIGroupInfo the wgpolicyk8s.io API group using the given getters.
func Build(polr, cpolr rest.Storage, scheme *runtime.Scheme, codecs serializer.CodecFactory) genericapiserver.APIGroupInfo {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(v1beta1.SchemeGroupVersion.Group, scheme, metav1.ParameterCodec, codecs)
	policyServerResources := map[string]rest.Storage{
		"policyreports":        polr,
		"clusterpolicyreports": cpolr,
	}
	apiGroupInfo.VersionedResourcesStorageMap[v1beta1.SchemeGroupVersion.Version] = policyServerResources

	return apiGroupInfo
}

// V1Beta1Install builds the metrics for the wgpolicyk8s.io/v1beta1 API, and then installs it into the given API policy-server.
func V1Beta1Install(store storage.Storage, server *genericapiserver.GenericAPIServer, scheme *runtime.Scheme, codecs serializer.CodecFactory) error {
	polr := PolicyReportStore(store)
	cpolr := ClusterPolicyReportStore(store)
	info := Build(polr, cpolr, scheme, codecs)
	return server.InstallAPIGroup(&info)
}
