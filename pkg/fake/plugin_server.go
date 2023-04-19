package fake

import (
	"context"
	"fmt"

	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const defaultDevicePath = "/dev/null"

var _ pluginapi.DevicePluginServer = &stub{}

func (m *stub) GetDevicePluginOptions(ctx context.Context, e *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{}, nil
}

func (m *stub) ListAndWatch(emtpy *pluginapi.Empty, stream pluginapi.DevicePlugin_ListAndWatchServer) error {
	klog.Info("Start ListAndWatch stream connection to Kubelet")

	resp := new(pluginapi.ListAndWatchResponse)
	for _, dev := range m.devices {
		resp.Devices = append(resp.Devices,
			&pluginapi.Device{ID: dev.ID, Health: dev.Health})
	}
	klog.InfoS("Send initial devices info", "response", resp)
	if err := stream.Send(resp); err != nil {
		klog.Errorf("cannot update device states: %s", err)
		m.stop <- true
		return err
	}

	for {
		select {
		case <-m.stop:
			return nil
		case <-m.devicesUpdated:
			resp := new(pluginapi.ListAndWatchResponse)
			for _, dev := range m.devices {
				resp.Devices = append(resp.Devices,
					&pluginapi.Device{ID: dev.ID, Health: dev.Health})
			}
			klog.InfoS("Send updated devices info", "response", resp)
			if err := stream.Send(resp); err != nil {
				klog.Errorf("cannot update device states: %s", err)
				m.stop <- true
				return err
			}
		}
	}
}

func (m *stub) Allocate(ctx context.Context, requests *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	klog.Info("No-op device allocation triggered by container creation")

	devices := make(map[string]pluginapi.Device, len(m.devices))
	for _, device := range m.devices {
		devices[device.ID] = *device
	}

	var responses pluginapi.AllocateResponse
	for _, req := range requests.ContainerRequests {
		response := &pluginapi.ContainerAllocateResponse{}
		env := make(map[string]string)

		for _, requestID := range req.DevicesIDs {
			dev, ok := devices[requestID]
			if !ok {
				return nil, fmt.Errorf("invalid allocation request with non-existing device %s", requestID)
			}
			if dev.Health != pluginapi.Healthy {
				return nil, fmt.Errorf("invalid allocation request with unhealthy device %s", requestID)
			}

			for key, val := range MakeEnv(resourceName, dev) {
				if vv, ok := env[key]; ok {
					env[key] = vv + "," + val
				} else {
					env[key] = val
				}
			}

			response.Devices = append(response.Devices, &pluginapi.DeviceSpec{
				HostPath:      defaultDevicePath,
				ContainerPath: defaultDevicePath,
				Permissions:   "rw",
			})
		}
		response.Envs = env
		responses.ContainerResponses = append(responses.ContainerResponses, response)
	}
	klog.Info("Send allocation responses to Kubelet")
	return &responses, nil
}

func (m *stub) PreStartContainer(ctx context.Context, r *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	klog.Errorf("PreStartContainer should NOT be called for fake device plugin")
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (m *stub) GetPreferredAllocation(ctx context.Context, r *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	klog.Errorf("GetPreferredAllocation should NOT be called for fake device plugin")
	return &pluginapi.PreferredAllocationResponse{}, nil
}
