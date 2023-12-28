package api

import (
	"context"

	generatedopenapi "github.com/kyverno/policy-server/pkg/api/generated/openapi"
	"github.com/kyverno/policy-server/pkg/storage"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1beta1"
)

type clusterpolicyreportsgetter struct {
	broadcaster   *watch.Broadcaster
	validator     validation.SchemaValidator
	listvalidator validation.SchemaValidator
	store         storage.Storage
}

const ClusterPolicyReportOpenApiV3SchemaKey = "ClusterPolicyReport"
const ClusterPolicyReportListOpenApiV3SchemaKey = "ClusterPolicyReportList"

func ClusterPolicyReportGetter(store storage.Storage) API {
	broadcaster := watch.NewBroadcaster(1000, watch.WaitIfChannelFull)
	openAPIDefinitions := generatedopenapi.GetOpenAPIDefinitions(refCallback)

	clusterPolicyReportOpenAPISchema := openAPIDefinitions[ClusterPolicyReportOpenApiV3SchemaKey]
	validator := validation.NewSchemaValidatorFromOpenAPI(&clusterPolicyReportOpenAPISchema.Schema)

	clusterPolicyReportListOpenAPISchema := openAPIDefinitions[ClusterPolicyReportListOpenApiV3SchemaKey]
	listvalidator := validation.NewSchemaValidatorFromOpenAPI(&clusterPolicyReportListOpenAPISchema.Schema)

	return &clusterpolicyreportsgetter{
		broadcaster:   broadcaster,
		validator:     validator,
		listvalidator: listvalidator,
		store:         store,
	}
}

func (c *clusterpolicyreportsgetter) New() runtime.Object {
	return &v1beta1.ClusterPolicyReport{}
}

func (c *clusterpolicyreportsgetter) Destroy() {
}

func (c *clusterpolicyreportsgetter) Kind() string {
	return "ClusterPolicyReport"
}

func (c *clusterpolicyreportsgetter) NewList() runtime.Object {
	return &v1beta1.ClusterPolicyReportList{}
}

func (c *clusterpolicyreportsgetter) Watch(ctx context.Context, _ *metainternalversion.ListOptions) (watch.Interface, error) {
	return c.broadcaster.Watch()
}

func (c *clusterpolicyreportsgetter) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1beta1.Table, error) {
	var table metav1beta1.Table

	switch t := object.(type) {
	case *v1beta1.ClusterPolicyReport:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		addClusterPolicyReportToTable(&table, *t)
	case *v1beta1.ClusterPolicyReportList:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		table.Continue = t.Continue
		addClusterPolicyReportToTable(&table, t.Items...)
	default:
	}

	return &table, nil
}

func (c *clusterpolicyreportsgetter) NamespaceScoped() bool {
	return true
}

func (c *clusterpolicyreportsgetter) GetSingularName() string {
	return "clusterpolicyreport"
}
