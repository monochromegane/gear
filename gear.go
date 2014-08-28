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

var listener net.Listener

func Start() {
	go serve()
	createPid()
	waitSignal()
}

func serve() {
	http.HandleFunc("/", dummyHandler)

	var err error
	var wg sync.WaitGroup
	conns := make(map[net.Conn]struct{})
	server := &http.Server{Addr: "0.0.0.0:8888"}
	server.ConnState = func(conn net.Conn, state http.ConnState) {
		fmt.Printf("State: %d\n", state)
		switch state {
		case http.StateActive:
			conns[conn] = struct{}{}
			wg.Add(1)
		case http.StateIdle, http.StateClosed:
			if _, exists := conns[conn]; exists {
				delete(conns, conn)
				wg.Done()
			}
		}
	}
	if os.Getenv("gear") == "" {
		listener, err = net.Listen("tcp", server.Addr)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		f := os.NewFile(3, "")
		listener, err = net.FileListener(f)
		parent := syscall.Getppid()
		syscall.Kill(parent, syscall.SIGTERM)
	}
	server.Serve(listener)
	wg.Wait()
	removeOldPid()
}

func createPid() {
	ioutil.WriteFile("gear.pid", []byte(strconv.Itoa(os.Getpid())), 0644)
}

func waitSignal() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGUSR2, syscall.SIGTERM)
	sig := <-ch
	fmt.Printf("Got a signal %v\n", sig)
	switch sig {
	case syscall.SIGTERM:
		stop()
	case syscall.SIGUSR2:
		restart()
	}
}

func restart() {
	renamePid()
	fork()
}

func renamePid() {
	os.Rename("gear.pid", "gear.pid.old")
}

func fork() {
	tl := listener.(*net.TCPListener)
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

func stop() {
	listener.Close()
}

func removeOldPid() {
	os.Remove("gear.pid.old")
}

func dummyHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello Gear from %d", os.Getpid())
}
