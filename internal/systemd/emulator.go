package systemd

import (
	"context"

	"github.com/cobaltcode-dev/kvm-node-agent/api/v1alpha1"
	"github.com/coreos/go-systemd/v22/dbus"
	logger "sigs.k8s.io/controller-runtime/pkg/log"
)

func NewSystemdEmulator(ctx context.Context) *InterfaceMock {
	log := logger.FromContext(ctx, "controller", "systemd-emulator")
	mockedInterface := &InterfaceMock{
		CloseFunc: func() {
			log.Info("CloseFunc called")
		},
		GetUnitByNameFunc: func(ctx context.Context, unit string) (dbus.UnitStatus, error) {
			log.Info("GetUnitByNameFunc called with with unit = " + unit)
			return dbus.UnitStatus{}, nil
		},
		IsConnectedFunc: func() bool {
			log.Info("GetUnitByNameFunc called")
			return true
		},
		ListUnitsByNamesFunc: func(ctx context.Context, units []string) ([]dbus.UnitStatus, error) {
			log.Info("GetUnitByNameFunc called")
			return nil, nil
		},
		ReconcileSysUpdateFunc: func(ctx context.Context, hv *v1alpha1.Hypervisor) (bool, error) {
			log.Info("GetUnitByNameFunc called")
			return true, nil
		},
		StartUnitFunc: func(ctx context.Context, unit string) (int, error) {
			log.Info("GetUnitByNameFunc called")
			return 0, nil
		},
	}
	return mockedInterface
}
