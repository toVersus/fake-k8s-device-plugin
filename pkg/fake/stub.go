package fake

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
	watcherapi "k8s.io/kubelet/pkg/apis/pluginregistration/v1"
)

const (
	resourceName = "3-shake.com/fake"
)

type stub struct {
	devices      map[string]*pluginapi.Device
	devicesMutex sync.Mutex

	socket string

	stop           chan bool
	wg             sync.WaitGroup
	devicesUpdated chan bool

	server *grpc.Server

	registrationStatus chan watcherapi.RegistrationStatus
	endpoint           string
}

func NewStub(socket string) *stub {
	return &stub{
		devices: map[string]*pluginapi.Device{
			"Dev-1": {ID: "Dev-1", Health: pluginapi.Healthy},
			"Dev-2": {ID: "Dev-2", Health: pluginapi.Healthy},
		},
		socket:             socket,
		stop:               make(chan bool),
		devicesUpdated:     make(chan bool),
		registrationStatus: make(chan watcherapi.RegistrationStatus),
		endpoint:           filepath.Base(socket),
	}
}

func (m *stub) cleanup() error {
	if err := os.Remove(m.socket); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (m *stub) Start() error {
	klog.Info("Cleanup existing unix socket for device-plugin server before starting up")
	if err := m.cleanup(); err != nil {
		return err
	}

	klog.Infof("Open unix domain socket for device-plugin server at %s", m.socket)
	sock, err := net.Listen("unix", m.socket)
	if err != nil {
		return err
	}

	m.wg.Add(1)
	m.server = grpc.NewServer([]grpc.ServerOption{}...)
	klog.Info("Register device-plugin service to gRPC server")
	pluginapi.RegisterDevicePluginServer(m.server, m)
	klog.Info("Register registration service to gRPC server")
	watcherapi.RegisterRegistrationServer(m.server, m)

	go func() {
		defer m.wg.Done()
		klog.Info("Start device-plugin server")
		m.server.Serve(sock)
	}()

	var lastDialErr error
	klog.Info("Wait for the device-plugin server to be ready and to accept a new request")
	wait.PollImmediate(1*time.Second, 10*time.Second, func() (bool, error) {
		klog.Info("Connection attempt to device-plugin server...")
		var conn *grpc.ClientConn
		conn, lastDialErr = dial(m.socket)
		if lastDialErr != nil {
			return false, nil
		}
		conn.Close()
		return true, nil
	})
	if lastDialErr != nil {
		return lastDialErr
	}

	klog.InfoS("Device-plugin server started successfuly", "socket", m.socket)
	return nil
}

func (m *stub) Stop() error {
	if m.server == nil {
		return nil
	}
	klog.Info("Shutdown device-plugin server")
	m.server.Stop()
	m.wg.Wait()
	m.server = nil
	close(m.stop)

	klog.Info("Cleanup existing unix socket for device-plugin server after shutting down")
	return m.cleanup()
}

// Register registers the device plugin for the given resourceName with Kubelet.
func (m *stub) Register() error {
	klog.InfoS("Create connection to Kubelet via unix socket", "socket", pluginapi.KubeletSocket)
	conn, err := dial(pluginapi.KubeletSocket)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     m.endpoint,
		ResourceName: resourceName,
		Options: &pluginapi.DevicePluginOptions{
			PreStartRequired:                false,
			GetPreferredAllocationAvailable: false,
		},
	}

	klog.Info("Register device-plugin server to Kubelet")
	_, err = client.Register(context.Background(), reqt)
	return err
}

// dial establishes the gRPC communication with the unit domain socket
func dial(unixSocketPath string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c, err := grpc.DialContext(ctx, unixSocketPath,
		grpc.WithAuthority("localhost"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", addr)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial unix socket: %s", err)
	}

	return c, nil
}
