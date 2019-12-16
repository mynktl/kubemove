package engine

import (
	"context"
	"strings"
	"time"

	"github.com/kubemove/kubemove/pkg/apis/kubemove/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//TODO
// new package for status

type resourceStatusFn func(*MoveEngineAction, unstructured.Unstructured) *v1alpha1.ResourceStatus

//TODO move to global action i.e postSync Action
var statusAction = map[string]resourceStatusFn{
	"deployment":            deploymentStatus,
	"persistentvolume":      pvStatus,
	"persistentvolumeclaim": pvcStatus,
	"namespace":             nsStatus,
}

func (m *MoveEngineAction) updateSyncStatus(obj unstructured.Unstructured) {
	rs := newResourceStatus(obj)
	rs.Phase = "Synced"

	kind := strings.ToLower(obj.GetKind())
	fn, ok := statusAction[kind]
	if ok {
		newRs := fn(m, obj)
		if newRs == nil {
			m.log.Error(nil, "Unable to update resourceStatus", "Resource", obj.GetKind(), "Name", obj.GetName())
		} else {
			rs = newRs
		}
	}
	m.addToSyncedResourceList(obj, *rs)
	return
}

func deploymentStatus(m *MoveEngineAction, obj unstructured.Unstructured) *v1alpha1.ResourceStatus {
	deploy := new(appsv1.Deployment)
	rs := newResourceStatus(obj)
	rs.Phase = "Synced"

	newObj, err := fetchRemoteResourceFromObj(m.remoteClient, obj)
	if err == nil {
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(newObj.UnstructuredContent(), deploy); err == nil {
			for _, d := range deploy.Status.Conditions {
				rs.Status = string(d.Type)
				rs.Reason = d.Reason
				break
			}
		} else {
			rs.Reason = err.Error()
		}
	} else {
		rs.Reason = err.Error()
	}
	return rs
}

func pvStatus(m *MoveEngineAction, obj unstructured.Unstructured) *v1alpha1.ResourceStatus {
	pv := new(v1.PersistentVolume)
	rs := newResourceStatus(obj)
	rs.Phase = "Synced"

	newObj, err := fetchRemoteResourceFromObj(m.remoteClient, obj)
	if err == nil {
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(newObj.UnstructuredContent(), pv); err == nil {
			rs.Status = string(pv.Status.Phase)
			rs.Reason = pv.Status.Reason
		} else {
			rs.Reason = err.Error()
		}
	} else {
		rs.Reason = err.Error()
	}
	return rs
}

func pvcStatus(m *MoveEngineAction, obj unstructured.Unstructured) *v1alpha1.ResourceStatus {
	pvc := new(v1.PersistentVolumeClaim)

	rs := newResourceStatus(obj)
	rs.Phase = "Synced"
	newObj, err := fetchRemoteResourceFromObj(m.remoteClient, obj)
	if err == nil {
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(newObj.UnstructuredContent(), pvc); err == nil {
			rs.Status = string(pvc.Status.Phase)
			for _, c := range pvc.Status.Conditions {
				rs.Reason = c.Reason
				break
			}
		} else {
			rs.Reason = err.Error()
		}

	} else {
		rs.Reason = err.Error()
	}

	return rs
}

func nsStatus(m *MoveEngineAction, obj unstructured.Unstructured) *v1alpha1.ResourceStatus {
	ns := new(v1.Namespace)
	rs := newResourceStatus(obj)
	rs.Phase = "Synced"

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), ns); err == nil {
		rs.Status = string(ns.Status.Phase)
	} else {
		rs.Reason = err.Error()
	}

	return rs
}

func newResourceStatus(obj unstructured.Unstructured) *v1alpha1.ResourceStatus {
	// TODO need to add clock at upper level, moveengineaction
	timestamp := time.Now()
	return &v1alpha1.ResourceStatus{
		Kind:       obj.GetKind(),
		Name:       obj.GetName(),
		SyncedTime: metav1.Time{Time: timestamp},
	}
}

func fetchRemoteResourceFromObj(cl client.Client, obj unstructured.Unstructured) (*unstructured.Unstructured, error) {
	newObj := &unstructured.Unstructured{}
	newObj.SetAPIVersion(obj.GetAPIVersion())
	newObj.SetKind(obj.GetKind())
	err := cl.Get(
		context.TODO(),
		client.ObjectKey{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		},
		newObj)
	return newObj, err
}
