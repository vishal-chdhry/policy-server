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
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/wg-policy-prototypes/policy-report/pkg/api/wgpolicyk8s.io/v1alpha2"
)

type cpolrStore struct {
	broadcaster *watch.Broadcaster
	store       storage.Storage
}

func ClusterPolicyReportStore(store storage.Storage) API {
	broadcaster := watch.NewBroadcaster(1000, watch.WaitIfChannelFull)

	return &cpolrStore{
		broadcaster: broadcaster,
		store:       store,
	}
}

func (c *cpolrStore) New() runtime.Object {
	return &v1alpha2.ClusterPolicyReport{}
}

func (c *cpolrStore) Destroy() {
}

func (c *cpolrStore) Kind() string {
	return "ClusterPolicyReport"
}

func (c *cpolrStore) NewList() runtime.Object {
	return &v1alpha2.ClusterPolicyReportList{}
}

func (c *cpolrStore) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	labelSelector := labels.Everything()
	if options != nil && options.LabelSelector != nil {
		labelSelector = options.LabelSelector
	}
	list, err := c.listCpolr()
	if err != nil {
		return &v1alpha2.ClusterPolicyReportList{}, errors.NewBadRequest("failed to list resource clusterpolicyreport")
	}

	// if labelSelector.String() == labels.Everything().String() {
	// 	return list, nil
	// }

	cpolrList := &v1alpha2.ClusterPolicyReportList{
		Items: make([]v1alpha2.ClusterPolicyReport, 0),
	}
	for _, cpolr := range list.Items {
		if cpolr.Labels == nil {
			return list, nil
		}
		if labelSelector.Matches(labels.Set(cpolr.Labels)) {
			cpolrList.Items = append(cpolrList.Items, cpolr)
		}
	}

	return cpolrList, nil
}

func (c *cpolrStore) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	report, err := c.getCpolr(name)
	if err != nil || report == nil {
		return &v1alpha2.ClusterPolicyReport{}, errors.NewNotFound(v1alpha2.Resource("clusterpolicyreports"), name)
	}
	return report, nil
}

func (c *cpolrStore) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
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
			return &v1alpha2.ClusterPolicyReport{}, err
		}
	}

	cpolr, ok := obj.(*v1alpha2.ClusterPolicyReport)
	if !ok {
		return &v1alpha2.ClusterPolicyReport{}, errors.NewBadRequest("failed to validate cluster policy report")
	}

	if !isDryRun {
		err := c.createCpolr(cpolr)
		if err != nil {
			return &v1alpha2.ClusterPolicyReport{}, errors.NewBadRequest(fmt.Sprintf("cannot create cluster policy report: %s", err.Error()))
		}
		c.broadcaster.Action(watch.Added, obj)
	}

	return obj, nil
}

func (c *cpolrStore) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	isDryRun := slices.Contains(options.DryRun, "All")

	if forceAllowCreate {
		oldObj, _ := c.getCpolr(name)
		updatedObject, _ := objInfo.UpdatedObject(ctx, oldObj)
		cpolr := updatedObject.(*v1alpha2.ClusterPolicyReport)
		c.updatePolr(cpolr, true)
		c.broadcaster.Action(watch.Added, updatedObject)
		return updatedObject, true, nil
	}

	oldObj, err := c.getCpolr(name)
	if err != nil {
		return &v1alpha2.ClusterPolicyReport{}, false, err
	}

	updatedObject, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil {
		return &v1alpha2.ClusterPolicyReport{}, false, err
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
			return &v1alpha2.ClusterPolicyReport{}, false, err
		}
	}

	cpolr, ok := updatedObject.(*v1alpha2.ClusterPolicyReport)
	if !ok {
		return &v1alpha2.ClusterPolicyReport{}, false, errors.NewBadRequest("failed to validate cluster policy report")
	}

	if !isDryRun {
		err := c.createCpolr(cpolr)
		if err != nil {
			return &v1alpha2.ClusterPolicyReport{}, false, errors.NewBadRequest(fmt.Sprintf("cannot create cluster policy report: %s", err.Error()))
		}
		c.broadcaster.Action(watch.Added, updatedObject)
	}

	return updatedObject, true, nil
}

func (c *cpolrStore) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	// TODO: Use propogation policy
	isDryRun := slices.Contains(options.DryRun, "All")

	cpolr, err := c.getCpolr(name)
	if err != nil {
		klog.ErrorS(err, "Failed to find cpolrs", "name", name)
		return &v1alpha2.ClusterPolicyReport{}, false, errors.NewNotFound(v1alpha2.Resource("clusterpolicyreports"), name)
	}

	err = deleteValidation(ctx, cpolr)
	if err != nil {
		klog.ErrorS(err, "invalid resource", "name", name)
		return &v1alpha2.ClusterPolicyReport{}, false, errors.NewBadRequest(fmt.Sprintf("invalid resource: %s", err.Error()))
	}

	if !isDryRun {
		err = c.deletePolr(cpolr)
		if err != nil {
			klog.ErrorS(err, "failed to delete cpolr", "name", name)
			return &v1alpha2.ClusterPolicyReport{}, false, errors.NewBadRequest(fmt.Sprintf("failed to delete clusterpolicyreport: %s", err.Error()))
		}
		c.broadcaster.Action(watch.Deleted, cpolr)
	}

	obj, err := c.cpolrToObj(cpolr)
	if err != nil {
		return nil, false, err
	}
	return obj, true, nil // TODO: Add protobuf in wgpolicygroup
}

func (c *cpolrStore) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions, listOptions *metainternalversion.ListOptions) (runtime.Object, error) {
	isDryRun := slices.Contains(options.DryRun, "All")

	obj, err := c.List(ctx, listOptions)
	if err != nil {
		klog.ErrorS(err, "Failed to find cpolrs")
		return &v1alpha2.ClusterPolicyReportList{}, errors.NewBadRequest("Failed to find cluster policy reports")
	}

	cpolrList, ok := obj.(*v1alpha2.ClusterPolicyReportList)
	if !ok {
		klog.ErrorS(err, "Failed to parse cpolrs")
		return &v1alpha2.ClusterPolicyReportList{}, errors.NewBadRequest("Failed to parse cluster policy reports")
	}

	if !isDryRun {
		for _, cpolr := range cpolrList.Items {
			obj, isDeleted, err := c.Delete(ctx, cpolr.GetName(), deleteValidation, options)
			if !isDeleted {
				klog.ErrorS(err, "Failed to delete cpolr", "name", cpolr.GetName())
				return &v1alpha2.ClusterPolicyReportList{}, errors.NewBadRequest(fmt.Sprintf("Failed to delete cluster policy report: %s", cpolr.GetName()))
			}
			c.broadcaster.Action(watch.Deleted, obj)
		}
	}
	return cpolrList, nil
}

func (c *cpolrStore) Watch(ctx context.Context, _ *metainternalversion.ListOptions) (watch.Interface, error) {
	return c.broadcaster.Watch()
}

func (c *cpolrStore) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1beta1.Table, error) {
	var table metav1beta1.Table

	switch t := object.(type) {
	case *v1alpha2.ClusterPolicyReport:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		addClusterPolicyReportToTable(&table, *t)
	case *v1alpha2.ClusterPolicyReportList:
		table.ResourceVersion = t.ResourceVersion
		table.SelfLink = t.SelfLink //nolint:staticcheck // keep deprecated field to be backward compatible
		table.Continue = t.Continue
		addClusterPolicyReportToTable(&table, t.Items...)
	default:
	}

	return &table, nil
}

func (c *cpolrStore) NamespaceScoped() bool {
	return false
}

func (c *cpolrStore) GetSingularName() string {
	return "clusterpolicyreport"
}

func (c *cpolrStore) ShortNames() []string {
	return []string{"cpolr"}
}

func (c *cpolrStore) key(name string) string {
	return fmt.Sprintf("/apis/%s/clusterpolicyreports/%s", v1alpha2.SchemeGroupVersion, name)
}

func (c *cpolrStore) keyForList() string {
	return fmt.Sprintf("/apis/%s/clusterpolicyreports/", v1alpha2.SchemeGroupVersion)
}

func (c *cpolrStore) cpolrToObj(cpolr *v1alpha2.ClusterPolicyReport) (runtime.Object, error) {
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

func (c *cpolrStore) getCpolr(name string) (*v1alpha2.ClusterPolicyReport, error) {
	var report v1alpha2.ClusterPolicyReport
	key := c.key(name)

	val, err := c.store.Get(context.TODO(), key)
	if err != nil {
		return nil, errorpkg.Wrapf(err, "could not find cluster policy report in store")
	}

	if err := json.Unmarshal(val.Data, &report); err != nil {
		return nil, errors.NewBadRequest("invalid object found")
	}

	return &report, nil
}

func (c *cpolrStore) listCpolr() (*v1alpha2.ClusterPolicyReportList, error) {
	key := c.keyForList()

	valList, err := c.store.List(context.TODO(), key, 0)
	if err != nil {
		return nil, errorpkg.Wrapf(err, "could not find cluster policy report in store")
	}

	reportList := &v1alpha2.ClusterPolicyReportList{
		Items: make([]v1alpha2.ClusterPolicyReport, len(valList)),
	}

	var cpolr v1alpha2.ClusterPolicyReport
	for i, val := range valList {
		if err := json.Unmarshal(val.Data, &cpolr); err != nil {
			return nil, errors.NewBadRequest("invalid object found")
		}
		reportList.Items[i] = cpolr
	}
	return reportList, nil
}

func (c *cpolrStore) createCpolr(report *v1alpha2.ClusterPolicyReport) error {
	key := c.key(report.Name)

	report.ResourceVersion = fmt.Sprint(1)
	report.UID = uuid.NewUUID()
	report.CreationTimestamp = metav1.Now()

	val, err := json.Marshal(report)
	if err != nil {
		return errorpkg.Wrapf(err, "could not marshal report")
	}
	return c.store.Update(context.TODO(), key, 1, val)
}

func (c *cpolrStore) updatePolr(report *v1alpha2.ClusterPolicyReport, force bool) error {
	key := c.key(report.GetName())
	if !force {
		oldReport, err := c.getCpolr(report.GetName())
		if err != nil {
			return errorpkg.Wrapf(err, "old cluster policy report not found")
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
	return c.store.Update(context.TODO(), key, rev, val)
}

func (c *cpolrStore) deletePolr(report *v1alpha2.ClusterPolicyReport) error {
	key := c.key(report.GetName())

	rev, err := strconv.ParseInt(report.ResourceVersion, 10, 64)
	if err != nil {
		return errorpkg.Wrapf(err, "could not marshal report's resource version")
	}
	return c.store.Delete(context.TODO(), key, rev)
}
