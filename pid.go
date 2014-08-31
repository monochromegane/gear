package gear

import (
	"io/ioutil"
	"os"
	"strconv"
)

func createPid() {
	ioutil.WriteFile("gear.pid", []byte(strconv.Itoa(os.Getpid())), 0644)
}

func renamePid() {
	os.Rename("gear.pid", "gear.pid.old")
}

func removeOldPid() {
	os.Remove("gear.pid.old")
}
