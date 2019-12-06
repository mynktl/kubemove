package engine

import (
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var annDynamicallyProvisioned = "pv.kubernetes.io/provisioned-by"

type pvFn func(interface{}, *MoveEngineAction, string, bool) (string, error)

var pvAction = map[string]pvFn{
	"persistentVolumeClaim": pvParsePVC,
}

func (m *MoveEngineAction) parsePodPV(pv interface{}, ns string, isSTS bool) error {
	//TODO
	// In spec.volume, there are two field, one is name and other one is volume details
	d, ok := pv.(map[string]interface{})
	if !ok {
		fmt.Printf("Failed to parse pod PV.. type is %T, expected Unstructured", pv)
		return errors.Errorf("Failed to parse pod PV.. type is %T, expected Unstructured", pv)
	}

	for k, v := range d {
		switch k {
		case "name":
		default:
			fn, ok := pvAction[k]
			if ok {
				pvName, err := fn(v, m, ns, isSTS)
				if err != nil {
					fmt.Printf("Failed to parse volumeSource for %v.. %v\n", k, err)
					continue
				}
				if len(pvName) > 0 {
					pvObj, err := m.getObj(pvName, "", "PersistentVolume")
					if err != nil {
						fmt.Printf("Failed to fetch PV %v.. %v\n", pvName, err)
					}
					if yes := isPVDynamicallyProvisioned(pvObj); yes {
						continue
					}
					if isSTS {
						m.addToSTSVolumeList(pvObj)
					} else {
						m.addToVolumeList(pvObj)
					}
				}
			}
		}
	}
	return nil
}

func pvParsePVC(o interface{}, m *MoveEngineAction, ns string, isSTS bool) (string, error) {
	pvc, ok := o.(map[string]interface{})
	if !ok {
		return "", errors.Errorf("Unexpected type of object")
	}
	val, ok, err := unstructured.NestedString(pvc, "claimName")
	if !ok || err != nil {
		return "", err
	}

	pvcObj, err := m.getObj(val, ns, "PersistentVolumeClaim")
	if err != nil {
		return "", err
	}

	if yes := m.ShouldRestore(pvcObj); yes {
		m.addToResourceList(pvcObj)
	}

	val, ok, err = unstructured.NestedString(pvcObj.Object, "spec", "volumeName")
	if !ok || err != nil {
		return "", err
	}

	return val, nil
}

func isPVDynamicallyProvisioned(obj unstructured.Unstructured) bool {
	ann := obj.GetAnnotations()
	if len(ann) == 0 {
		return false
	}

	for k, v := range ann {
		//TODO
		if k == annDynamicallyProvisioned && len(v) != 0 {
			return true
		}
	}
	return false
}
