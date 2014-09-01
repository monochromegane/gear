package gear

import (
	"net/http"
	"os"
)

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
