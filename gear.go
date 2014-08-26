package gear

import (
	"fmt"
	"net"
	"net/http"
)

func Start() {

	http.HandleFunc("/", dummyHandler)

	server := &http.Server{Addr: "0.0.0.0:8888"}
	l, err := net.Listen("tcp", server.Addr)
	if err != nil {
		fmt.Println(err)
		return
	}
	server.Serve(l)
}

func dummyHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello Gear!")
}
