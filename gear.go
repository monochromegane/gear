package gear

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
)

type GearServer struct {
	wg       sync.WaitGroup
	server   *http.Server
	listener net.Listener
}

func ListenAndServe(addr string, handler http.Handler) error {
	server := newServer(addr, handler)
	err := server.ListenAndServe()
	return err
}

func newServer(addr string, handler http.Handler) *GearServer {
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	gear := &GearServer{
		server: server,
	}
	return gear
}

func (g *GearServer) ListenAndServe() error {
	listener, err := g.Listener()
	if err != nil {
		return err
	}
	go g.Serve(listener)
	createPid()

	g.Wait()
	return nil
}

func (g *GearServer) Listener() (net.Listener, error) {
	if g.listener != nil {
		return g.listener, nil
	}
	var err error
	var l net.Listener
	if isParentProcess() {
		l, err = net.Listen("tcp", g.server.Addr)
	} else {
		f := os.NewFile(3, "")
		l, err = net.FileListener(f)
	}
	g.listener = l
	return l, err
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

	if isChildProcess() {
		parent := syscall.Getppid()
		syscall.Kill(parent, syscall.SIGTERM)
	}
	err := g.server.Serve(l)
	if err != nil {
		return err
	}
	g.wg.Wait()
	removeOldPid()
	return nil
}

func (g *GearServer) Wait() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGUSR2, syscall.SIGTERM)
	sig := <-ch
	fmt.Printf("Got a signal %v\n", sig)
	switch sig {
	case syscall.SIGTERM:
		g.listener.Close()
	case syscall.SIGUSR2:
		g.restart()
	}
}

func (g GearServer) restart() {
	renamePid()
	g.fork()
}

func (g GearServer) fork() {
	tl := g.listener.(*net.TCPListener)
	fl, _ := tl.File()
	cmd := exec.Command(os.Args[0])
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = []string{"gear=child"}
	cmd.ExtraFiles = []*os.File{fl}
	err := cmd.Start()
	if err != nil {
		fmt.Printf("start err: %s\n", err)
	}
}

func isParentProcess() bool {
	return os.Getenv("gear") == ""
}

func isChildProcess() bool {
	return !isParentProcess()
}

func createPid() {
	ioutil.WriteFile("gear.pid", []byte(strconv.Itoa(os.Getpid())), 0644)
}

func renamePid() {
	os.Rename("gear.pid", "gear.pid.old")
}

func removeOldPid() {
	os.Remove("gear.pid.old")
}
