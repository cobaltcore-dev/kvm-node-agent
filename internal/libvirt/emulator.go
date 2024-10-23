package libvirt

import (
	"context"

	"github.com/cobaltcode-dev/kvm-node-agent/api/v1alpha1"
	logger "sigs.k8s.io/controller-runtime/pkg/log"
)

func NewLibVirtEmulator(ctx context.Context) *InterfaceMock {
	log := logger.FromContext(ctx, "controller", "libvirt-emulator")
	mockedInterface := &InterfaceMock{
		CloseFunc: func() error {
			log.Info("CloseFunc called")
			return nil
		},
		ConnectFunc: func() error {
			log.Info("Connect Func called")
			return nil
		},
		GetInstancesFunc: func() ([]v1alpha1.Instance, error) {
			log.Info("GetInstancesFunc Func called")
			return nil, nil
		},
		GetVersionFunc: func() (string, error) {
			log.Info("GetVersionFunc Func called")
			return "10.9.0", nil
		},
		IsConnectedFunc: func() bool {
			log.Info("IsConnectedFunc Func called")
			return true
		},
	}
	return mockedInterface
}
