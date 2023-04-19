package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/toVersus/fake-k8s-device-plugin/pkg/fake"
	"k8s.io/klog/v2"
)

const (
	socketDir = "/var/lib/kubelet/device-plugins"
)

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()
	klog.Info("Fake device-plugin started")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	m := fake.NewStub(socketDir + "/fake.sock")
	if err := m.Start(); err != nil {
		klog.Fatalf("Failed to start fake device-plugin server: %s", err)
	}

	if err := m.Register(); err != nil {
		klog.Fatalf("Failed to register fake device-plugin to kubelet: %s", err)
	}

	<-sig
	klog.Info("Got termination signal, fake device-plugin stopped")
	m.Stop()
}
