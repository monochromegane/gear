package gear

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"syscall"
)

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
