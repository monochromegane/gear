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

func NewServer(addr string, handler http.Handler) *Server {
	return &Server{
		server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
		process: process{
			ppid: os.Getppid(),
			pid:  os.Getpid(),
			env:  os.Getenv("gear"),
		},
	}
}
