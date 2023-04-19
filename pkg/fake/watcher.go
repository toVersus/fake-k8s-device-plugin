package fake

import (
	"context"

	"k8s.io/klog/v2"

	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
	watcherapi "k8s.io/kubelet/pkg/apis/pluginregistration/v1"
)

var _ watcherapi.RegistrationServer = &stub{}

func (m *stub) GetInfo(ctx context.Context, req *watcherapi.InfoRequest) (*watcherapi.PluginInfo, error) {
	klog.Info("GetInfo() called by kubelet")
	return &watcherapi.PluginInfo{
		Type:              watcherapi.DevicePlugin,
		Name:              resourceName,
		Endpoint:          m.endpoint,
		SupportedVersions: []string{pluginapi.Version},
	}, nil
}

func (m *stub) NotifyRegistrationStatus(ctx context.Context, status *watcherapi.RegistrationStatus) (*watcherapi.RegistrationStatusResponse, error) {
	klog.Info("NotifyRegistrationStatus() called by kubelet")
	if m.registrationStatus != nil {
		m.registrationStatus <- *status
	}
	klog.Infof("Registration status: %v", status)
	if !status.PluginRegistered {
		klog.Errorf("Registration failed: %s", status.Error)
	}
	return &watcherapi.RegistrationStatusResponse{}, nil
}
