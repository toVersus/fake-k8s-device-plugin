package fake

import (
	"strings"

	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// MakeEnv creates the environment variable in the format: <resource-prefix>=<device_id>,<device_id>,...
func MakeEnv(resourceName string, dev pluginapi.Device) map[string]string {
	key := resourceName
	key = strings.Map(func(r rune) rune {
		if r == '.' || r == '/' {
			return '_'
		}
		return r
	}, key)
	key = strings.ToUpper(key)

	val := dev.ID

	klog.InfoS("Generate environment variables to pass to the container", "key", key, "val", val)
	return map[string]string{
		key: val,
	}
}
