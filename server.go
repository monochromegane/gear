package gear

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type GearServer struct {
	wg      sync.WaitGroup
	server  *http.Server
	process process
}

func (g *GearServer) ListenAndServe() error {
	listener, err := g.Listener()
	if err != nil {
		return err
	}

	go g.WaitSignal(listener)

	createPid()
	g.Serve(listener)

	return nil
}

func (g *GearServer) Listener() (net.Listener, error) {
	if g.process.isFirst() {
		return net.Listen("tcp", g.server.Addr)
	} else {
		f := os.NewFile(3, "")
		return net.FileListener(f)
	}
}

func (g *GearServer) WaitSignal(l net.Listener) {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGUSR2, syscall.SIGTERM)
	sig := <-ch
	fmt.Printf("Got a signal %v\n", sig)
	switch sig {
	case syscall.SIGTERM:
		l.Close()
	case syscall.SIGUSR2:
		g.process.forkWithListener(l)
	}
}

func (g *GearServer) Serve(l net.Listener) error {
	g.server.ConnState = func(conn net.Conn, state http.ConnState) {
		fmt.Printf("State: %d\n", state)
		switch state {
		case http.StateNew:
			g.wg.Add(1)
		case http.StateHijacked, http.StateClosed:
			g.wg.Done()
		}
	}

	g.process.stopParent()

	err := g.server.Serve(l)
	if err != nil {
		return err
	}
	g.wg.Wait()
	removeOldPid()
	return nil
}
