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

	server := &http.Server{Addr: "0.0.0.0:8888"}
	var err error
	if os.Getenv("gear") == "" {
		listener, err = net.Listen("tcp", server.Addr)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		f := os.NewFile(3, "")
		listener, err = net.FileListener(f)
	}
	server.Serve(listener)
}

func createPid() {
	ioutil.WriteFile("gear.pid", []byte(strconv.Itoa(os.Getpid())), 0644)
}

func waitSignal() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGUSR2)
	for sig := range ch {
		fmt.Printf("Got a signal %v", sig)
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
	cmd.Start()
}

func dummyHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello Gear!")
}
