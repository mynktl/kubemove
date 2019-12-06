package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/go-logr/logr"
	"github.com/kubemove/kubemove/pkg/apis/kubemove/v1alpha1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
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

type MultiResource struct {
	groupResource []string
	seen          bool
}

type NamedObj map[string]unstructured.Unstructured

type MoveEngineAction struct {
	mov                v1alpha1.MoveEngine
	selector           labels.Selector
	client             client.Client
	dclient            *discovery.DiscoveryClient
	mapper             meta.RESTMapper
	remoteMapper       meta.RESTMapper
	remoteClient       client.Client
	remotedClient      *discovery.DiscoveryClient
	remotedyClient     dynamic.Interface
	multiAPIResources  map[string]*MultiResource
	resourcesMap       map[schema.GroupVersionResource]NamedObj
	resourceList       []unstructured.Unstructured
	exposedResourceMap map[MResources]unstructured.Unstructured
	volMap             map[MResources]unstructured.Unstructured
	stsVolMap          map[MResources]unstructured.Unstructured
	syncedResourceMap  map[MResources]v1alpha1.ResourceStatus
	syncedVolMap       map[MResources]v1alpha1.VolumeStatus
	namespace          string
	plugin             string
	log                logr.Logger
}

// If any API resource have multiple APIVersion then add it here
func NewMultiAPIResources() map[string]*MultiResource {
	return map[string]*MultiResource{
		"daemonsets": &MultiResource{
			groupResource: []string{"extensions", "apps"},
			seen:          false,
		},

		"deployments": &MultiResource{
			groupResource: []string{"extensions", "apps"},
			seen:          false,
		},

		"ingresses": &MultiResource{
			groupResource: []string{"extensions", "networking.k8s.io"},
			seen:          false,
		},

		"networkpolicies": &MultiResource{
			groupResource: []string{"extensions", "networking.k8s.io"},
			seen:          false,
		},

		"podsecuritypolicies": &MultiResource{
			groupResource: []string{"extensions", "policy"},
			seen:          false,
		},

		"replicasets": &MultiResource{
			groupResource: []string{"extensions", "apps"},
			seen:          false,
		},
	}

}
func NewMoveEngineAction(log logr.Logger, c client.Client) *MoveEngineAction {
	sr := make(map[schema.GroupVersionResource]NamedObj)
	er := make(map[MResources]unstructured.Unstructured)
	v := make(map[MResources]unstructured.Unstructured)
	sv := make(map[MResources]unstructured.Unstructured)
	return &MoveEngineAction{
		log:                log,
		client:             c,
		resourcesMap:       sr,
		exposedResourceMap: er,
		volMap:             v,
		stsVolMap:          sv,
		multiAPIResources:  NewMultiAPIResources(),
	}
}

func (m *MoveEngineAction) ParseResourceEngine(mov *v1alpha1.MoveEngine) error {
	if len(mov.Spec.MovePair) != 0 {
		/*
			pairObj, err := mpair.Get(mov.Spec.MovePair, m.client)
			if err != nil {
				fmt.Printf("Failed to fetch movePair %v.. %v\n", mov.Spec.MovePair, err)
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
		return err
	}

	//	m.dumpSyncResourceList()
	//	m.dumpVolList()
	//	m.dumpSTSVolList()
	return nil
}

func (m *MoveEngineAction) UpdateSyncResourceList() error {
	rlist, err := m.getAPIResources()
	if err != nil {
		return err
	}

	for _, g := range rlist {
		k, ok := m.multiAPIResources[g.Name]
		if ok {
			if k.seen {
				continue
			}
			k.seen = true
		}
		err = m.parseAPIResource(g)
		if err != nil {
			m.log.Error(err, "syncing %v errored %v", g)
		}
	}

	fmt.Printf("\n\nCreating resources..\n")
	for _, g := range rlist {
		// To create reources at remote cluster
		err = m.CreateResourceAtRemote(g)
		if err != nil {
			fmt.Printf("Failed to create resource.. %v\n", err)
		}
	}

	return nil
}

func (m *MoveEngineAction) CreateResourceAtRemote(api metav1.APIResource) error {
	gvr := schema.GroupVersionResource{
		Group:    api.Group,
		Version:  api.Version,
		Resource: strings.ToLower(api.Kind),
	}

	objMap, ok := m.resourcesMap[gvr]
	if !ok {
		return nil
	}

	for _, v := range objMap {
		if m.isResourceSynced(v) {
			continue
		}

		if err := m.RUpdateObject(v); err != nil {
			fmt.Printf("Failed to update object %v/%v/%v %v\n", v.GetAPIVersion(), v.GetKind(), v.GetName(), err)
			continue
		}

		if err := m.RCreateObject(v); err != nil {
			fmt.Printf("Failed to create object %v/%v/%v %v\n", v.GetAPIVersion(), v.GetKind(), v.GetName(), err)
			continue
		}
	}
	return nil
}

func (m *MoveEngineAction) parseAPIResource(api metav1.APIResource) error {
	//TODO move it to fn
	//TODO pass top groupVersion
	switch api.Name {
	case "leases", "nodes", "events":
		fmt.Printf("Skipping %v\n", api.Name)
		return nil
	}

	// check if ns exists or not
	if api.Name == "namespaces" {
		if err := m.parseNamespace(api); err != nil {
			fmt.Printf("Namespace sync failed %v\n", err)
			return err
		}
	}
	if len(m.mov.Spec.Namespace) != 0 && api.Namespaced {
		if err := m.parseResourceList(api); err != nil {
			fmt.Printf("Failed to create resourceList for %v %v.. %v\n", api.Name, api.Group, err)
			return err
		}
	}

	if !api.Namespaced {
		switch api.Kind {
		case "StorageClass":
			if err := m.parseResourceList(api); err != nil {
				fmt.Printf("Failed to create resourceList for %v %v.. %v\n", api.Name, api.Group, err)
			}
		}
	}
	return nil
}

func (m *MoveEngineAction) parseNamespace(api metav1.APIResource) error {
	if len(m.mov.Spec.Namespace) != 0 &&
		len(m.mov.Spec.RemoteNamespace) != 0 {
		obj, err := m.getResource(api, client.ObjectKey{Name: m.mov.Spec.Namespace})
		if err != nil {
			fmt.Printf("Failed to fetch resource %v\n", err)
			return err
		}
		obj.SetName(m.mov.Spec.RemoteNamespace)
		m.addToResourceList(obj)
	} else {
		list, err := m.ListResources(api)
		if err != nil {
			fmt.Printf("Failed to fetch list for %v %v.. %v\n", api.Name, api.Group, err)
			return err
		}
		for _, i := range list.Items {
			m.addToResourceList(i)
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

func (m *MoveEngineAction) parseResourceList(api metav1.APIResource) error {
	list, err := m.ListResources(api)
	if err != nil {
		fmt.Printf("Failed to fetch list for %v %v.. %v\n", api.Name, api.Group, err)
		return err
	}

	for _, l := range list.Items {
		if err := m.parseResource(api, l); err != nil {
			fmt.Printf("Failed to parse resource for %v %v.. %v\n", api.Name, api.Group, err)
			return err
		}
	}
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
	if !m.ShouldRestore(obj) {
		fmt.Printf("Skipping APIResource %v %v\n", api.Name, obj.GetName())
		// lets add this to synced resource list
		m.addToSyncedResourceList(obj)
		return nil
	}

	m.addToSyncedResourceList(obj)

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

	ns := ""
	if api.Namespaced {
		ns = m.mov.Spec.Namespace
	}

	err := m.client.List(
		context.TODO(),
		&client.ListOptions{
			Namespace:     ns,
			LabelSelector: m.selector,
		},
		list)
	return list, err

}

func (m *MoveEngineAction) getResource(api metav1.APIResource, key client.ObjectKey) (unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   api.Group,
		Version: api.Version,
		Kind:    api.Kind,
	})

	err := m.client.Get(
		context.TODO(),
		key,
		obj)
	return *obj, err

}

func (m *MoveEngineAction) ShouldRestore(obj unstructured.Unstructured) bool {
	shouldRestore := true

	switch obj.GetKind() {
	case "Pod", "ReplicaSet":
		//TODO
		// This will be removed by OwnerReference check
		or := obj.GetOwnerReferences()
		for _, o := range or {
			sr := newMResourceFromOR(o)
			if _, ok := m.getFromResourceList(sr); ok {
				fmt.Printf("%v/%v already created by %v/%v\n", obj.GetKind(), obj.GetName(), o.Kind, o.Name)
				shouldRestore = false
				break
			} else if o.Kind == "ReplicaSet" {
				shouldRestore = m.ShouldRestoreRS(o.Name, obj.GetNamespace())
			} else {
				// STS or deployment is not added to syncResource list
				// Let's add it
				ro, err := m.getObj(o.Name, obj.GetNamespace(), o.Kind)
				if err != nil {
					//TODO check if not exist
					fmt.Printf("Failed to fetch %v/%v/%v.. %v\n", obj.GetNamespace(), o.Kind, o.Name, err)
					continue
				}
				//TODO refactor
				sr := newMResourceFromObj(ro)
				if _, ok := m.getFromResourceList(sr); !ok {
					m.addToResourceList(ro)
				}
				shouldRestore = false
			}
		}
	case "Node", "Event":
		shouldRestore = false
	case "Service", "Endpoints":
		m.addToExposedResourceList(obj)
		//TODO check if switch call
		shouldRestore = false
	}

	return shouldRestore
}

func (m *MoveEngineAction) parseResource(api metav1.APIResource, obj unstructured.Unstructured) error {
	switch api.Name {
	case "pods":
		if err := m.parseVolumes(api, obj, m.checkIfSTSPod(obj)); err != nil {
			fmt.Printf("Failed to parse volumes for %v/%v\n", obj.GetKind(), obj.GetName())
		}
	}

	if !m.ShouldRestore(obj) {
		fmt.Printf("Skipping api %v %v\n", api.Name, obj.GetName())
		return nil
	}

	m.addToResourceList(obj)
	return nil
}

func (m *MoveEngineAction) dumpSyncResourceList() {
	fmt.Printf("Resources which will be synced to remote cluster:\n")
	for _, l := range m.resourceList {
		fmt.Printf("%v %v %v\n", l.GetAPIVersion(), l.GetKind(), l.GetName())
	}
}

func (m *MoveEngineAction) dumpVolList() {
	fmt.Printf("Volumes which will be created at remote cluster\n")
	for _, l := range m.volMap {
		fmt.Printf("%v %v %v\n", l.GetAPIVersion(), l.GetKind(), l.GetName())
	}
	return
}

func (m *MoveEngineAction) dumpSTSVolList() {
	fmt.Printf("Volumes which will be created by STS at remote cluster\n")
	for _, l := range m.stsVolMap {
		fmt.Printf("%v %v %v\n", l.GetAPIVersion(), l.GetKind(), l.GetName())
	}
	return
}

func (m *MoveEngineAction) parseVolumes(api metav1.APIResource, obj unstructured.Unstructured, isSTS bool) error {
	if api.Name != "pods" {
		return nil
	}

	p, ok, err := unstructured.NestedFieldCopy(obj.Object, "spec", "volumes")
	if !ok && err == nil {
		fmt.Printf("No volumes for %v/%v\n", obj.GetKind(), obj.GetName())
		return nil
	}
	if err != nil {
		fmt.Printf("Failed to get volumes for %v/%v.. %v\n", obj.GetKind(), obj.GetName(), err)
		return err
	}

	pvlist, ok := p.([]interface{})
	if !ok {
		fmt.Printf("Failed to parse volume list for %v/%v.. type is %T, expected []interface{}\n", obj.GetKind(), obj.GetName(), p)
		return errors.Errorf("Failed to parse volumes for %v/%v.. type is %T, expected []interface{}\n", obj.GetKind(), obj.GetName(), p)
	}
	for _, l := range pvlist {
		err = m.parsePodPV(l, obj.GetNamespace(), isSTS)
		if err != nil {
			fmt.Printf("Failed to parse pod PV.. %v\n", err)
		}
	}
	return nil
}

func (m *MoveEngineAction) checkIfSTSPod(obj unstructured.Unstructured) bool {
	or := obj.GetOwnerReferences()
	for _, o := range or {
		if o.Kind == "StatefulSet" {
			return true
		}
	}
	return false
}

func convertJSONToUnstructured(data []byte) (unstructured.Unstructured, error) {
	obj := unstructured.Unstructured{}

	om := make(map[string]interface{})
	err := json.Unmarshal(data, &om)
	if err != nil {
		return obj, err
	}

	obj.Object = om
	return obj, nil
}

func (m *MoveEngineAction) ShouldRestoreRS(name, ns string) bool {
	//TODO add check from cmd line
	shouldRestore := true
	if obj, err := m.getObj(name, ns, "ReplicaSet"); err == nil {
		shouldRestore = m.ShouldRestore(obj)
		if shouldRestore {
			m.addToResourceList(obj)
		}
		shouldRestore = false
	}
	return shouldRestore
}

func (m *MoveEngineAction) getObj(name, ns, kind string) (unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}

	// TODO check ResourceFor : if returns top one
	gvr, err := m.mapper.ResourcesFor(schema.ParseGroupResource(kind).WithVersion(""))
	if err != nil {
		fmt.Printf("Failed to fetch resource for %v.. %v\n", kind, err)
		return *obj, nil
	}

	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gvr[0].Group,
		Version: gvr[0].Version,
		Kind:    kind,
	})
	err = m.client.Get(
		context.TODO(),
		client.ObjectKey{Name: name, Namespace: ns},
		obj)
	return *obj, err
}
