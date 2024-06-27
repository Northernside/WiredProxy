package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"wirednode/node"
)

func main() {
	node.Run(getFileHash())
}

func getFileHash() string {
	self, err := os.Executable()
	if err != nil {
		log.Fatalln("Error getting executable path:", err)
	}

	cmd := exec.Command("sha256sum", self)
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	// format output (remove every backslash)
	// and split by space

	hash := strings.Split(strings.ReplaceAll(string(out), "\\", ""), " ")[0]
	return hash
}
