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

type policyreportsgetter struct {
	broadcaster   *watch.Broadcaster
	validator     validation.SchemaValidator
	listvalidator validation.SchemaValidator
	store         storage.Storage
}

const PolicyReportOpenApiV3SchemaKey = "PolicyReport"
const PolicyReportListOpenApiV3SchemaKey = "PolicyReportList"

func PolicyReportGetter(store storage.Storage) API {
	broadcaster := watch.NewBroadcaster(1000, watch.WaitIfChannelFull)
	openAPIDefinitions := generatedopenapi.GetOpenAPIDefinitions(refCallback)

	policyReportOpenAPISchema := openAPIDefinitions[PolicyReportOpenApiV3SchemaKey]
	validator := validation.NewSchemaValidatorFromOpenAPI(&policyReportOpenAPISchema.Schema)

	policyReportListOpenAPISchema := openAPIDefinitions[PolicyReportListOpenApiV3SchemaKey]
	listvalidator := validation.NewSchemaValidatorFromOpenAPI(&policyReportListOpenAPISchema.Schema)

	return &policyreportsgetter{
		broadcaster:   broadcaster,
		validator:     validator,
		listvalidator: listvalidator,
		store:         store,
	}
}

func (p *policyreportsgetter) New() runtime.Object {
	return &v1beta1.PolicyReport{}
}

func (p *policyreportsgetter) Destroy() {
}

func (p *policyreportsgetter) Kind() string {
	return "PolicyReport"
}

func (p *policyreportsgetter) NewList() runtime.Object {
	return &v1beta1.PolicyReportList{}
}

func (p *policyreportsgetter) Watch(ctx context.Context, _ *metainternalversion.ListOptions) (watch.Interface, error) {
	return p.broadcaster.Watch()
}

func (p *policyreportsgetter) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1beta1.Table, error) {
	var table metav1beta1.Table

	switch t := object.(type) {
	case *v1beta1.PolicyReport:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		addPolicyReportToTable(&table, *t)
	case *v1beta1.PolicyReportList:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		table.Continue = t.Continue
		addPolicyReportToTable(&table, t.Items...)
	default:
	}

	return &table, nil
}

func (p *policyreportsgetter) NamespaceScoped() bool {
	return false
}

func (p *policyreportsgetter) GetSingularName() string {
	return "policyreport"
}
