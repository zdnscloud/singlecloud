package handler

import (
	"reflect"
	"testing"

	ut "github.com/zdnscloud/cement/unittest"

	"github.com/zdnscloud/singlecloud/pkg/types"
)

func getAdvancedOpts(podOwner interface{}) types.AdvancedOptions {
	return reflect.ValueOf(podOwner).Elem().FieldByName("AdvancedOptions").Interface().(types.AdvancedOptions)
}

func TestPodOwnerReflect(t *testing.T) {
	opts := []types.AdvancedOptions{
		types.AdvancedOptions{
			ExposedServiceType: "udp",
		},
		types.AdvancedOptions{
			ReloadWhenConfigChange: false,
		},
		types.AdvancedOptions{
			ExposedServices: []types.ExposedService{
				types.ExposedService{
					ContainerPortName: "udp_port",
					ServicePort:       53,
					IngressPort:       33,
				},
			},
		},
	}

	for _, opt := range opts {
		deploy := &types.Deployment{
			AdvancedOptions: opt,
		}
		ut.Equal(t, opt, getAdvancedOpts(deploy))

		statefulSet := &types.StatefulSet{
			AdvancedOptions: opt,
		}
		ut.Equal(t, opt, getAdvancedOpts(statefulSet))

		daemonSet := &types.DaemonSet{
			AdvancedOptions: opt,
		}
		ut.Equal(t, opt, getAdvancedOpts(daemonSet))
	}
}
