package main

import (
	"fmt"

	"github.com/kubemove/kubemove/pkg/apis/kubemove/v1alpha1"
	"github.com/kubemove/kubemove/pkg/engine"
	kmpair "github.com/kubemove/kubemove/pkg/pair"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var remoteCfg = "/tmp/cluster-dest"
var destCfg = "/tmp/cluster-dest"

var log = logf.Log.WithName("test")

// MOV dummy
var MOV = &v1alpha1.MoveEngine{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "testMove",
		Namespace: "movens",
	},
	Spec: v1alpha1.MoveEngineSpec{
		MovePair:         "testPair",
		Namespace:        "default",
		RemoteNamespace:  "testns",
		Mode:             "backup",
		PluginProvider:   "testPlugin",
		IncludeResources: false,
	},
}

var mlabel = "app=minio"

func main() {
	ls, err := metav1.ParseToLabelSelector(mlabel)
	if err != nil {
		fmt.Printf("Failed to parse label %v\n", err)
		return
	}
	MOV.Spec.Selectors = ls

	err = loadMPAIR()
	if err != nil {
		fmt.Printf("Failed to load MPAIR %v\n", err)
		return
	}

	_, err = tverifyMovePairStatus(&engine.MPAIR)
	if err != nil {
		fmt.Printf("Failed to verify mpair status %v\n", err)
		return
	}

	mgr, err := LoadClient()
	if err != nil {
		log.Error(err, "Failed to load client")
		return
	}

	me := engine.NewMoveEngineAction(log, mgr.GetClient())
	err = me.ParseResource(MOV)
	if err != nil {
		log.Error(err, "Failed to parse moveEngine")
		return
	}
	return
}

func loadMPAIR() error {
	config, err := clientcmd.LoadFromFile(remoteCfg)
	if err != nil {
		fmt.Printf("Failed to load config from %v.. %v\n", remoteCfg, err)
		return err
	}
	engine.MPAIR.Name = "testPair"
	engine.MPAIR.Namespace = "kubens"
	engine.MPAIR.Spec.Config = *config
	return nil
}

func LoadClient() (manager.Manager, error) {
	// TODO update the path
	cfg, err := clientcmd.BuildConfigFromFlags("", "/home/.kube/config")
	if err != nil {
		fmt.Printf("Failed to fetch k8s cluster config. %+v", err)
		return nil, err
	}

	manager, err := manager.New(cfg, manager.Options{
		Namespace: "default",
	})

	return manager, err
}

func tverifyMovePairStatus(mpair *v1alpha1.MovePair) (string, error) {
	err := clientcmd.Validate(mpair.Spec.Config)
	if err != nil {
		return "Errored", err
	}

	client, err := kmpair.FetchPairDiscoveryClient(mpair)
	if err != nil {
		return "Errored", err
	}

	// To verify access, let's fetch remote server version
	_, err = client.ServerVersion()
	if err != nil {
		return "Errored", err
	}
	return "success", nil
}
