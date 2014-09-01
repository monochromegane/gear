package gear

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

type GearServer struct {
	wg      sync.WaitGroup
	server  *http.Server
	process process
}

type process struct {
	ppid int
	pid  int
	env  string
}

func (p process) isFirst() bool {
	return p.ppid == 1 || p.env == ""
}

func (p process) isForked() bool {
	return !p.isFirst()
}

func (p process) stopParent() {
	if p.isFirst() {
		return
	}
	syscall.Kill(p.ppid, syscall.SIGTERM)
}

func (p process) forkWithListener(l net.Listener) {
	// Get file from net.Listener
	fl, err := l.(*net.TCPListener).File()
	if err != nil {
		fmt.Printf("err in forkWithListener %v\n", err)
	}

	// Fork own process
	cmd := exec.Command(os.Args[0])
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = []string{"gear=child"}
	cmd.ExtraFiles = []*os.File{fl}
	err = cmd.Start()
	if err != nil {
		fmt.Printf("start err: %s\n", err)
	}
}

func ListenAndServe(addr string, handler http.Handler) error {
	server := NewServer(addr, handler)
	err := server.ListenAndServe()
	return err
}

func NewServer(addr string, handler http.Handler) *GearServer {
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	gear := &GearServer{
		server: server,
		process: process{
			ppid: os.Getppid(),
			pid:  os.Getpid(),
			env:  os.Getenv("gear"),
		},
	}
	return gear
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
	conns := make(map[net.Conn]struct{})
	g.server.ConnState = func(conn net.Conn, state http.ConnState) {
		fmt.Printf("State: %d\n", state)
		switch state {
		case http.StateActive:
			conns[conn] = struct{}{}
			g.wg.Add(1)
		case http.StateIdle, http.StateClosed:
			if _, exists := conns[conn]; exists {
				delete(conns, conn)
				g.wg.Done()
			}
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
