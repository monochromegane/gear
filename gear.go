package gear

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func Start() {
	go serve()
	waitSignal()
}

func serve() {
	http.HandleFunc("/", dummyHandler)

	server := &http.Server{Addr: "0.0.0.0:8888"}
	l, err := net.Listen("tcp", server.Addr)
	if err != nil {
		fmt.Println(err)
		return
	}
	server.Serve(l)
}

func waitSignal() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGUSR2)
	for sig := range ch {
		fmt.Printf("Got a signal %v", sig)
	}
}

func dummyHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello Gear!")
}
