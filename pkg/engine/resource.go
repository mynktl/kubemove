package engine

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/go-logr/logr"
	"github.com/kubemove/kubemove/pkg/apis/kubemove/v1alpha1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO test
var MPAIR v1alpha1.MovePair

type MResources struct {
	Name       string
	Kind       string
	APIVersion string
}

type MoveEngineAction struct {
	mov              v1alpha1.MoveEngine
	selector         labels.Selector
	client           client.Client
	dclient          *discovery.DiscoveryClient
	remoteClient     client.Client
	remotedClient    *discovery.DiscoveryClient
	remotedyClient   dynamic.Interface
	syncedResources  map[MResources]unstructured.Unstructured
	exposedResources map[MResources]unstructured.Unstructured
	namespace        string
	plugin           string
	volumes          []MResources
	log              logr.Logger
}

func NewMoveEngineAction(log logr.Logger, c client.Client) *MoveEngineAction {
	sr := make(map[MResources]unstructured.Unstructured)
	er := make(map[MResources]unstructured.Unstructured)
	return &MoveEngineAction{
		log:              log,
		client:           c,
		syncedResources:  sr,
		exposedResources: er,
	}
}

func (m *MoveEngineAction) ParseResource(mov *v1alpha1.MoveEngine) error {
	if len(mov.Spec.MovePair) != 0 {
		/*
			pairObj, err := mpair.Get(mov.Spec.MovePair, m.client)
			if err != nil {
				fmt.Printf("In update resou ask\n")
				return err
			}
		*/
		pairObj := MPAIR
		if err := m.updateClient(&pairObj); err != nil {
			return errors.New("MovePair not defined")
		}
	}

	ls, err := metav1.LabelSelectorAsSelector(mov.Spec.Selectors)
	if err != nil {
		fmt.Printf("Failed to parse label selector %v\n", err)
		return err
	}

	m.mov = *mov
	m.selector = ls
	err = m.UpdateSyncResourceList()
	if err != nil {
		fmt.Printf("got error %v\n", err)
		return err
	}
	return nil
}

func (m *MoveEngineAction) UpdateSyncResourceList() error {
	rlist, err := m.getAPIResources()
	if err != nil {
		return err
	}

	for _, g := range rlist {
		err := m.SyncResourceList(g)
		if err != nil {
			m.log.Error(err, "syncing %v errored %v", g)
		}
	}
	return nil
}

func (m *MoveEngineAction) SyncResourceList(api metav1.APIResource) error {
	//TODO move it to fn
	if api.Name == "leases" {
		fmt.Printf("Skipping %v\n", api.Name)
		return nil
	}
	// check if ns exists or not
	if api.Name == "namespaces" {
		if err := m.CheckORCreateNS(api); err != nil {
			fmt.Printf("Namespace sync failed %v\n", err)
			return err
		}
	}
	if len(m.mov.Spec.Namespace) != 0 && api.Namespaced {
		if err := m.CreateResourceList(api); err != nil {
			fmt.Printf("Failed to create resourceList for %v %v.. %v\n", api.Name, api.Group, err)
			return err
		}
	}

	if !api.Namespaced {
		if api.Kind == "StorageClass" {
			if err := m.CreateResourceList(api); err != nil {
				fmt.Printf("Failed to create resourceList for %v %v.. %v\n", api.Name, api.Group, err)
			}
		}
	}
	return nil
}

func (m *MoveEngineAction) CheckORCreateNS(api metav1.APIResource) error {
	// check if ns exists or not
	list, err := m.ListResources(api)
	if err != nil {
		fmt.Printf("Failed to fetch list for %v %v.. %v\n", api.Name, api.Group, err)
		return err
	}

	for _, i := range list.Items {
		if len(m.mov.Spec.Namespace) != 0 &&
			len(m.mov.Spec.RemoteNamespace) != 0 &&
			//			m.mov.Spec.Namespace == m.mov.Spec.RemoteNamespace &&
			m.mov.Spec.Namespace == i.GetName() {
			i.SetName(m.mov.Spec.RemoteNamespace)
			if err := m.CreateResource(api, i); err != nil {
				if !k8serror.IsAlreadyExists(err) {
					fmt.Printf("Failed to create %v:%v %v\n", api.Name, i.GetName(), err)
					return err
				}
			}
		}
	}
	return nil
}

func (m *MoveEngineAction) CreateRemotePV() error {
	return nil
}

func (m *MoveEngineAction) CreateRemoteResources() error {
	return nil
}

func (m *MoveEngineAction) DestroyExposedResources() error {
	return nil
}

func (m *MoveEngineAction) CreateResourceList(api metav1.APIResource) error {
	list, err := m.ListResources(api)
	if err != nil {
		fmt.Printf("Failed to fetch list for %v %v.. %v\n", api.Name, api.Group, err)
		return err
	}

	for _, l := range list.Items {
		if err := m.CreateResource(api, l); err != nil {
			if k8serror.IsAlreadyExists(err) {
				//TODO update the resource
				return nil
			}
			return err
		}
	}
	return nil
}

func (m *MoveEngineAction) CreateResource(api metav1.APIResource, obj unstructured.Unstructured) error {
	if !m.ShouldRestore(api, obj) {
		fmt.Printf("Skipping api %v %v\n", api.Name, obj.GetName())
		// lets add this to synced resource list
		sr := MResources{
			Name:       obj.GetName(),
			Kind:       obj.GetKind(),
			APIVersion: obj.GetAPIVersion(),
		}
		m.syncedResources[sr] = obj
		return nil
	}

	m.UpdateObject(api, obj)
	sr := MResources{
		Name:       obj.GetName(),
		Kind:       obj.GetKind(),
		APIVersion: obj.GetAPIVersion(),
	}
	m.syncedResources[sr] = obj

	if _, err := m.remotedyClient.
		Resource(schema.GroupVersionResource{
			Group:    api.Group,
			Version:  api.Version,
			Resource: api.Name,
		}).
		Namespace(obj.GetNamespace()).
		Create(&obj, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func (m *MoveEngineAction) ListResources(api metav1.APIResource) (*unstructured.UnstructuredList, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   api.Group,
		Version: api.Version,
		Kind:    api.Kind,
	})

	err := m.client.List(
		context.TODO(),
		&client.ListOptions{
			Namespace:     m.mov.Spec.Namespace,
			LabelSelector: m.selector,
		},
		list)
	return list, err

}

func (m *MoveEngineAction) UpdateObject(api metav1.APIResource, obj unstructured.Unstructured) {
	unstructured.RemoveNestedField(obj.Object, "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(obj.Object, "metadata", "resourceVersion")
	unstructured.RemoveNestedField(obj.Object, "metadata", "selfLink")

	if api.Namespaced {
		if len(m.mov.Spec.RemoteNamespace) != 0 {
			obj.SetNamespace(m.mov.Spec.RemoteNamespace)
		}
	}

	switch api.Name {
	case "services":
		//TODO Handle this separately
		val, found, err := unstructured.NestedString(obj.Object, "spec", "clusterIP")
		if err == nil && found && val != "None" {
			_ = unstructured.SetNestedField(obj.Object, "", "spec", "clusterIP")
		}
	}

}

func (m *MoveEngineAction) ShouldRestore(api metav1.APIResource, obj unstructured.Unstructured) bool {
	shouldRestore := true
	switch api.Name {
	case "pods":
		fallthrough
	case "replicasets":
		or := obj.GetOwnerReferences()
		for _, o := range or {
			sr := MResources{
				Name:       o.Name,
				Kind:       o.Kind,
				APIVersion: o.APIVersion,
			}
			if _, ok := m.syncedResources[sr]; ok {
				fmt.Printf("%v/%v %v already created by %v/%v\n", api.Group, api.Name, obj.GetName(), o.Kind, o.Name)
				shouldRestore = false
				break
			}
		}
	case "nodes":
		fallthrough
	case "events":
		shouldRestore = false
	}

	return shouldRestore
}
