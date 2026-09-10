package server

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hwcer/cosrpc"
	rpcxclient "github.com/smallnest/rpcx/client"
	rpcxserver "github.com/smallnest/rpcx/server"
)

type closeTestRegister struct {
	stopErr error
	stops   atomic.Int32
}

func (*closeTestRegister) Start() error { return nil }

func (*closeTestRegister) Register(string, any, string) error { return nil }

func (r *closeTestRegister) Stop() error {
	r.stops.Add(1)
	return r.stopErr
}

func TestCloseTimesOutInflightRequestAndStopsRegister(t *testing.T) {
	oldTimeout := cosrpc.Config.Timeout
	cosrpc.Config.Timeout = 0
	t.Cleanup(func() { cosrpc.Config.Timeout = oldTimeout })

	xs := New()
	xs.started.Store(1)
	registerErr := errors.New("register stop failed")
	reg := &closeTestRegister{stopErr: registerErr}
	xs.register = reg

	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseHandler := func() {
		releaseOnce.Do(func() { close(release) })
	}
	t.Cleanup(releaseHandler)
	t.Cleanup(func() { _ = xs.Close() })
	xs.Server.AddHandler("close-test", "block", func(*rpcxserver.Context) error {
		close(entered)
		<-release
		return nil
	})

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- xs.Server.Serve("tcp", "127.0.0.1:0")
	}()

	deadline := time.Now().Add(2 * time.Second)
	for xs.Server.Address() == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if xs.Server.Address() == nil {
		t.Fatal("rpcx server did not start")
	}

	client := rpcxclient.NewClient(rpcxclient.DefaultOption)
	if err := client.Connect("tcp", xs.Server.Address().String()); err != nil {
		t.Fatalf("connect rpcx server: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	callDone := make(chan *rpcxclient.Call, 1)
	client.Go(context.Background(), "close-test", "block", struct{}{}, &struct{}{}, callDone)
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("rpc handler did not start")
	}

	err := xs.Close()
	releaseHandler()

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Close() error = %v, want context deadline exceeded", err)
	}
	if !errors.Is(err, registerErr) {
		t.Errorf("Close() error = %v, want register stop error", err)
	}
	if got := reg.stops.Load(); got != 1 {
		t.Errorf("register Stop() calls = %d, want 1", got)
	}
	if err := xs.Close(); err != nil {
		t.Errorf("second Close() error = %v, want nil", err)
	}
	if got := reg.stops.Load(); got != 1 {
		t.Errorf("register Stop() calls after second Close() = %d, want 1", got)
	}

	select {
	case <-serveDone:
	case <-time.After(2 * time.Second):
		t.Fatal("rpcx server did not stop")
	}
}
