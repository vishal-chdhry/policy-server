package api

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"

	"github.com/kyverno/policy-server/pkg/storage"
	errorpkg "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/watch"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

type polrStore struct {
	broadcaster *watch.Broadcaster
	store       storage.Storage
}

func PolicyReportStore(store storage.Storage) API {
	broadcaster := watch.NewBroadcaster(1000, watch.WaitIfChannelFull)

	return &polrStore{
		broadcaster: broadcaster,
		store:       store,
	}
}

func (p *polrStore) New() runtime.Object {
	return &v1alpha2.PolicyReport{}
}

func (p *polrStore) Destroy() {
}

func (p *polrStore) Kind() string {
	return "PolicyReport"
}

func (p *polrStore) NewList() runtime.Object {
	return &v1alpha2.PolicyReportList{}
}

func (p *polrStore) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	labelSelector := labels.Everything()
	if options != nil && options.LabelSelector != nil {
		labelSelector = options.LabelSelector
	}
	namespace := genericapirequest.NamespaceValue(ctx)
	list, err := p.listPolr(namespace)
	if err != nil {
		return &v1alpha2.PolicyReportList{}, errors.NewBadRequest("failed to list resource policyreport")
	}

	// if labelSelector == labels.Everything() {
	// 	return list, nil
	// }

	polrList := &v1alpha2.PolicyReportList{
		Items: make([]v1alpha2.PolicyReport, 0),
	}
	for _, polr := range list.Items {
		if polr.Labels == nil {
			return list, nil
		}
		if labelSelector.Matches(labels.Set(polr.Labels)) {
			polrList.Items = append(polrList.Items, polr)
		}
	}

	return polrList, nil
}

func (p *polrStore) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	namespace := genericapirequest.NamespaceValue(ctx)
	report, err := p.getPolr(name, namespace)
	if err != nil || report == nil {
		return &v1alpha2.PolicyReport{}, errors.NewNotFound(v1alpha2.Resource("policyreports"), name)
	}
	return report, nil
}

func (p *polrStore) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	isDryRun := slices.Contains(options.DryRun, "All")

	err := createValidation(ctx, obj)
	if err != nil {
		switch options.FieldValidation {
		case "Ignore":
		case "Warn":
			// return &admissionv1.AdmissionResponse{
			// 	Allowed:  false,
			// 	Warnings: []string{err.Error()},
			// }, nil
		case "Strict":
			return &v1alpha2.PolicyReport{}, err
		}
	}

	polr, ok := obj.(*v1alpha2.PolicyReport)
	if !ok {
		return &v1alpha2.PolicyReport{}, errors.NewBadRequest("failed to validate policy report")
	}

	namespace := genericapirequest.NamespaceValue(ctx)
	if len(polr.Namespace) == 0 {
		polr.Namespace = namespace
	}

	if !isDryRun {
		err := p.createPolr(polr)
		if err != nil {
			return &v1alpha2.PolicyReport{}, errors.NewBadRequest(fmt.Sprintf("cannot create policy report: %s", err.Error()))
		}
		p.broadcaster.Action(watch.Added, obj)
	}

	return obj, nil
}

func (p *polrStore) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	isDryRun := slices.Contains(options.DryRun, "All")
	namespace := genericapirequest.NamespaceValue(ctx)

	if forceAllowCreate {
		oldObj, _ := p.getPolr(name, namespace)
		updatedObject, _ := objInfo.UpdatedObject(ctx, oldObj)
		polr := updatedObject.(*v1alpha2.PolicyReport)
		p.updatePolr(polr, true)
		p.broadcaster.Action(watch.Added, updatedObject)
		return updatedObject, true, nil
	}

	oldObj, err := p.getPolr(name, namespace)
	if err != nil {
		return &v1alpha2.PolicyReport{}, false, err
	}

	updatedObject, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil {
		return &v1alpha2.PolicyReport{}, false, err
	}
	err = updateValidation(ctx, updatedObject, oldObj)
	if err != nil {
		switch options.FieldValidation {
		case "Ignore":
		case "Warn":
			// return &admissionv1.AdmissionResponse{
			// 	Allowed:  false,
			// 	Warnings: []string{err.Error()},
			// }, nil
		case "Strict":
			return &v1alpha2.PolicyReport{}, false, err
		}
	}

	polr, ok := updatedObject.(*v1alpha2.PolicyReport)
	if !ok {
		return &v1alpha2.PolicyReport{}, false, errors.NewBadRequest("failed to validate policy report")
	}

	if len(polr.Namespace) == 0 {
		polr.Namespace = namespace
	}

	if !isDryRun {
		err := p.updatePolr(polr, false)
		if err != nil {
			return &v1alpha2.PolicyReport{}, false, errors.NewBadRequest(fmt.Sprintf("cannot create policy report: %s", err.Error()))
		}
		p.broadcaster.Action(watch.Added, updatedObject)
	}

	return updatedObject, true, nil
}

func (p *polrStore) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	// TODO: Use propogation policy
	isDryRun := slices.Contains(options.DryRun, "All")
	namespace := genericapirequest.NamespaceValue(ctx)

	polr, err := p.getPolr(name, namespace)
	if err != nil {
		klog.ErrorS(err, "Failed to find polrs", "name", name, "namespace", klog.KRef("", namespace))
		return &v1alpha2.PolicyReport{}, false, errors.NewNotFound(v1alpha2.Resource("policyreports"), name)
	}

	err = deleteValidation(ctx, polr)
	if err != nil {
		klog.ErrorS(err, "invalid resource", "name", name, "namespace", klog.KRef("", namespace))
		return &v1alpha2.PolicyReport{}, false, errors.NewBadRequest(fmt.Sprintf("invalid resource: %s", err.Error()))
	}

	if !isDryRun {
		err = p.deletePolr(polr)
		if err != nil {
			klog.ErrorS(err, "failed to delete polr", "name", name, "namespace", klog.KRef("", namespace))
			return &v1alpha2.PolicyReport{}, false, errors.NewBadRequest(fmt.Sprintf("failed to delete policyreport: %s", err.Error()))
		}
		p.broadcaster.Action(watch.Deleted, polr)
	}

	obj, err := p.polrToObj(polr)
	if err != nil {
		return nil, false, err
	}
	return obj, true, nil // TODO: Add protobuf in wgpolicygroup
}

func (p *polrStore) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions, listOptions *metainternalversion.ListOptions) (runtime.Object, error) {
	isDryRun := slices.Contains(options.DryRun, "All")
	namespace := genericapirequest.NamespaceValue(ctx)

	obj, err := p.List(ctx, listOptions)
	if err != nil {
		klog.ErrorS(err, "Failed to find polrs", "namespace", klog.KRef("", namespace))
		return &v1alpha2.PolicyReportList{}, errors.NewBadRequest("Failed to find policy reports")
	}

	polrList, ok := obj.(*v1alpha2.PolicyReportList)
	if !ok {
		klog.ErrorS(err, "Failed to parse polrs", "namespace", klog.KRef("", namespace))
		return &v1alpha2.PolicyReportList{}, errors.NewBadRequest("Failed to parse policy reports")
	}

	if !isDryRun {
		for _, polr := range polrList.Items {
			obj, isDeleted, err := p.Delete(ctx, polr.GetName(), deleteValidation, options)
			if !isDeleted {
				klog.ErrorS(err, "Failed to delete polr", "name", polr.GetName(), "namespace", klog.KRef("", namespace))
				return &v1alpha2.PolicyReportList{}, errors.NewBadRequest(fmt.Sprintf("Failed to delete policy report: %s/%s", polr.Namespace, polr.GetName()))
			}
			p.broadcaster.Action(watch.Deleted, obj)
		}
	}
	return polrList, nil
}

func (p *polrStore) Watch(ctx context.Context, _ *metainternalversion.ListOptions) (watch.Interface, error) {
	return p.broadcaster.Watch()
}

func (p *polrStore) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1beta1.Table, error) {
	var table metav1beta1.Table

	switch t := object.(type) {
	case *v1alpha2.PolicyReport:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		addPolicyReportToTable(&table, *t)
	case *v1alpha2.PolicyReportList:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		table.Continue = t.Continue
		addPolicyReportToTable(&table, t.Items...)
	default:
	}

	return &table, nil
}

func (p *polrStore) NamespaceScoped() bool {
	return true
}

func (p *polrStore) GetSingularName() string {
	return "policyreport"
}

func (p *polrStore) ShortNames() []string {
	return []string{"polr"}
}

func (p *polrStore) key(name, namespace string) string {
	return fmt.Sprintf("/apis/%s/namespaces/%s/policyreports/%s", v1alpha2.SchemeGroupVersion, namespace, name)
}

func (p *polrStore) keyForList(namespace string) string {
	return fmt.Sprintf("/apis/%s/namespaces/%s/policyreports/", v1alpha2.SchemeGroupVersion, namespace)
}

func (c *polrStore) polrToObj(cpolr *v1alpha2.PolicyReport) (runtime.Object, error) {
	var unst unstructured.Unstructured
	var bytes []byte
	var err error
	if bytes, err = json.Marshal(cpolr); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(bytes, &unst); err != nil {
		return nil, err
	}
	return unst.DeepCopyObject(), nil
}

func (p *polrStore) getPolr(name, namespace string) (*v1alpha2.PolicyReport, error) {
	var report v1alpha2.PolicyReport
	key := p.key(name, namespace)

	val, err := p.store.Get(context.TODO(), key)
	if err != nil {
		return nil, errorpkg.Wrapf(err, "could not find policy report in store")
	}

	if err := json.Unmarshal(val.Data, &report); err != nil {
		return nil, errors.NewBadRequest("invalid object found")
	}

	return &report, nil
}

func (p *polrStore) listPolr(namespace string) (*v1alpha2.PolicyReportList, error) {
	key := p.keyForList(namespace)

	valList, err := p.store.List(context.TODO(), key, 0)
	if err != nil {
		return nil, errorpkg.Wrapf(err, "could not find policy report in store")
	}

	reportList := &v1alpha2.PolicyReportList{
		Items: make([]v1alpha2.PolicyReport, len(valList)),
	}

	var polr v1alpha2.PolicyReport
	for i, val := range valList {
		if err := json.Unmarshal(val.Data, &polr); err != nil {
			return nil, errors.NewBadRequest("invalid object found")
		}
		reportList.Items[i] = polr
	}
	return reportList, nil
}

func (p *polrStore) createPolr(report *v1alpha2.PolicyReport) error {
	key := p.key(report.Name, report.Namespace)

	report.ResourceVersion = fmt.Sprint(1)
	report.UID = uuid.NewUUID()
	report.CreationTimestamp = metav1.Now()

	val, err := json.Marshal(report)
	if err != nil {
		return errorpkg.Wrapf(err, "could not marshal report")
	}
	return p.store.Create(context.TODO(), key, val)
}

func (p *polrStore) updatePolr(report *v1alpha2.PolicyReport, force bool) error {
	key := p.key(report.Name, report.Namespace)
	if !force {
		oldReport, err := p.getPolr(report.GetName(), report.Namespace)
		if err != nil {
			return errorpkg.Wrapf(err, "old policy report not found")
		}
		oldRV, err := strconv.ParseInt(oldReport.ResourceVersion, 10, 64)
		if err != nil {
			return errorpkg.Wrapf(err, "could not parse resource version")
		}

		report.ResourceVersion = fmt.Sprint(oldRV + 1)
	} else {
		report.ResourceVersion = "1"
	}
	val, err := json.Marshal(report)
	if err != nil {
		return errorpkg.Wrapf(err, "could not marshal report")
	}

	rev, _ := strconv.ParseInt(report.ResourceVersion, 10, 64)
	return p.store.Update(context.TODO(), key, rev, val)
}

func (p *polrStore) deletePolr(report *v1alpha2.PolicyReport) error {
	key := p.key(report.Name, report.Namespace)

	rev, err := strconv.ParseInt(report.ResourceVersion, 10, 64)
	if err != nil {
		return errorpkg.Wrapf(err, "could not marshal report's resource version")
	}
	return p.store.Delete(context.TODO(), key, rev)
}
