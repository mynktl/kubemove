package engine

import (
	"fmt"

	"github.com/kubemove/kubemove/pkg/apis/kubemove/v1alpha1"
	pair "github.com/kubemove/kubemove/pkg/pair"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/restmapper"
)

var topresource = []string{
	"customresourcedefinitions",
	"namespaces",
	"storageclasses",
	"serviceaccounts",
	"customresourcedefinitions",
	"secrets",
	"configmaps",
	"persistentvolumes",
	"persistentvolumeclaims",
	"limitranges",
	"statefulsets",
	"deployments",
	"daemonsets",
	"replicaset",
	"pods",
}

func (m *MoveEngineAction) updateClient(mpair *v1alpha1.MovePair) error {
	var err error

	if m.dclient, err = pair.FetchDiscoveryClient(); err != nil {
		return err
	}

	gr, err := restmapper.GetAPIGroupResources(m.dclient)
	if err != nil {
		fmt.Printf("Failed to fetch group resources %v\n", err)
		return err
	}

	m.mapper = restmapper.NewDiscoveryRESTMapper(gr)

	if m.remoteClient, err = pair.FetchPairClient(mpair); err != nil {
		return err
	}

	if m.remotedClient, err = pair.FetchPairDiscoveryClient(mpair); err != nil {
		return err
	}

	rgr, err := restmapper.GetAPIGroupResources(m.remotedClient)
	if err != nil {
		fmt.Printf("Failed to fetch remote group resources %v\n", err)
		return err
	}
	m.remoteMapper = restmapper.NewDiscoveryRESTMapper(rgr)

	if m.remotedyClient, err = pair.FetchPairDynamicClient(mpair); err != nil {
		return err
	}
	return nil
}

func (m *MoveEngineAction) getAPIResources() ([]metav1.APIResource, error) {
	tgvr := []schema.GroupVersionResource{}
	tagvr := make(map[schema.GroupVersionResource]metav1.APIResource)
	for _, t := range topresource {
		// TODO need to check error
		gvr, err := m.mapper.ResourcesFor(schema.ParseGroupResource(t).WithVersion(""))
		if err != nil {
			return nil, err
		}
		for _, g := range gvr {
			tagvr[g] = metav1.APIResource{}
			tgvr = append(tgvr, g)
		}
	}

	//TODO check for failed API groups
	pr, _ := m.dclient.ServerPreferredResources()
	d := discovery.FilteredBy(
		discovery.ResourcePredicateFunc(
			func(g string, r *metav1.APIResource) bool {
				return discovery.SupportsAllVerbs{Verbs: []string{"list", "create", "get", "delete"}}.Match(g, r)
			}),
		pr,
	)

	toplist := []metav1.APIResource{}
	restlist := []metav1.APIResource{}
	for _, resourceGroup := range d {
		gv, err := schema.ParseGroupVersion(resourceGroup.GroupVersion)
		if err != nil {
			fmt.Printf("unable to parse GroupVersion %s.. %v", resourceGroup.GroupVersion, err)
			return nil, errors.Wrapf(err, "unable to parse GroupVersion %s", resourceGroup.GroupVersion)
		}

		for _, resource := range resourceGroup.APIResources {
			resource.Group = gv.Group
			resource.Version = gv.Version
			if _, v := tagvr[gv.WithResource(resource.Name)]; v {
				tagvr[gv.WithResource(resource.Name)] = resource
				continue
			}
			restlist = append(restlist, resource)
		}
	}

	for _, k := range tgvr {
		toplist = append(toplist, tagvr[k])
	}

	return append(toplist, restlist...), nil
}
