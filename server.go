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

type Server struct {
	wg      sync.WaitGroup
	server  *http.Server
	process process
}

func (s *Server) ListenAndServe() error {
	listener, err := s.Listener()
	if err != nil {
		return err
	}

	go s.WaitSignal(listener)

	createPid()
	s.Serve(listener)

	return nil
}

func (s *Server) Listener() (net.Listener, error) {
	if s.process.isFirst() {
		return net.Listen("tcp", s.server.Addr)
	} else {
		f := os.NewFile(3, "")
		return net.FileListener(f)
	}
}

func (s *Server) WaitSignal(l net.Listener) {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGUSR2, syscall.SIGTERM)
	for {
		sig := <-ch
		fmt.Printf("Got a signal %v on %d\n", sig, s.process.pid)
		switch sig {
		case syscall.SIGTERM:
			signal.Stop(ch)
			l.Close()
		case syscall.SIGUSR2:
			s.process.forkWithListener(l)
		}
	}
}

func (s *Server) Serve(l net.Listener) error {
	s.server.ConnState = func(conn net.Conn, state http.ConnState) {
		fmt.Printf("State: %d\n", state)
		switch state {
		case http.StateNew:
			s.wg.Add(1)
		case http.StateHijacked, http.StateClosed:
			s.wg.Done()
		}
	}

	s.process.stopParent()

	err := s.server.Serve(l)
	if err != nil {
		return err
	}
	fmt.Printf("waiting graceful shutdown on %d\n", s.process.pid)
	s.wg.Wait()
	removeOldPid()
	return nil
}
