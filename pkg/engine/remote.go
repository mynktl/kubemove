package engine

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (m *MoveEngineAction) RUpdateObject(obj unstructured.Unstructured) error {
	unstructured.RemoveNestedField(obj.Object, "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(obj.Object, "metadata", "resourceVersion")
	unstructured.RemoveNestedField(obj.Object, "metadata", "selfLink")
	unstructured.RemoveNestedField(obj.Object, "spec", "status")

	if len(obj.GetNamespace()) != 0 {
		if len(m.mov.Spec.RemoteNamespace) != 0 {
			obj.SetNamespace(m.mov.Spec.RemoteNamespace)
		}
	}

	switch obj.GetKind() {
	case "Service":
		//TODO Handle this separately
		val, found, err := unstructured.NestedString(obj.Object, "spec", "clusterIP")
		if err == nil && found && val != "None" {
			_ = unstructured.SetNestedField(obj.Object, "", "spec", "clusterIP")
		}
	case "PersistentVolumeClaim":
		unstructured.RemoveNestedField(obj.Object, "metadata", "annotations")
		unstructured.RemoveNestedField(obj.Object, "spec", "volumeName")
	case "PersistentVolume":
		unstructured.RemoveNestedField(obj.Object, "metadata", "annotations")
		unstructured.RemoveNestedField(obj.Object, "spec", "claimRef")
	}
	return nil
}

func (m *MoveEngineAction) RCreateObject(obj unstructured.Unstructured) error {
	//TODO check
	var err error
	var gvr schema.GroupVersionResource

	gvrlist, err := m.remoteMapper.ResourcesFor(schema.ParseGroupResource(obj.GetKind()).WithVersion(""))
	if err == nil && len(gvrlist) != 0 {
		gvr = gvrlist[0]
	} else {
		gv, _ := schema.ParseGroupVersion(obj.GetAPIVersion())
		gvr = schema.GroupVersionResource{
			Group:    gv.Group,
			Version:  gv.Version,
			Resource: obj.GetKind(),
		}
	}

	_, err = m.remotedyClient.
		Resource(gvr).
		Namespace(obj.GetNamespace()).
		Create(&obj, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("Object created %v/%v/%v.. %v\n", obj.GetAPIVersion(), obj.GetKind(), obj.GetName(), err)
		return err
	}
	fmt.Printf("Object created %v/%v/%v\n", obj.GetAPIVersion(), obj.GetKind(), obj.GetName())
	m.addToSyncedResourceList(obj)
	//TODO if volume or PVC need to append in list
	return nil
}
