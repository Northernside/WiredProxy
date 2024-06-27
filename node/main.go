package main

import (
	"log"
	"os"
	"os/exec"
	"fmt"
	"strings"
	"wirednode/node"

	"wired.rip/wiredutils/config"
	"wired.rip/wiredutils/terminal"
)

// go:embed wirednode.service
var wiredService string

func main() {
	args := os.Args[1:]

	config.Init()
	log.SetFlags(0)
	prefix := fmt.Sprintf("%s.%s » ", config.GetSystemKey(), config.GetWiredHost())
	log.SetPrefix(terminal.PrefixColor + prefix + terminal.Reset)

	switch args[0] {
	case "start":
		node.Run(getFileHash())
	case "install":
		systemdInstall()
	}
}

func systemdInstall() {
	// check if system is linux
	if os.Getenv("OS") != "linux" {
		log.Fatalln("Systemd installation is only available on linux")
	}

	// check if systemd is available
	cmd := exec.Command("systemctl", "--version")
	err := cmd.Run()
	if err != nil {
		log.Fatalln("Systemd is not available on this system")
	}

	// check if user has permission to write to /etc/systemd/system
	_, err = os.Create("/etc/systemd/system/wirednode.service")
	if err != nil {
		log.Fatalln("Permission denied. Run as root")
	}

	// write service file
	err = os.WriteFile("/etc/systemd/system/wirednode.service", formatServiceFile(), 0644)
	if err != nil {
		log.Fatalln("Error writing service file:", err)
	}

	cmd = exec.Command("systemctl", "enable", "--now", "/etc/systemd/system/wirednode.service")
	err = cmd.Run()
	if err != nil {
		log.Fatalln("Error enabling service:", err)
	}

	cmd = exec.Command("systemctl", "start", "wirednode")
	err = cmd.Run()
	if err != nil {
		log.Fatalln("Error starting service:", err)
	}
	
	log.Println("Service installed and started")
}

func formatServiceFile() []byte {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	bin, err := os.Executable()
	if err != nil {
		panic(err)
	}

	wiredService = strings.ReplaceAll(wiredService, "{WORKINGDIR}", dir)
	wiredService = strings.ReplaceAll(wiredService, "{BINPATH}", bin)
	wiredService = strings.ReplaceAll(wiredService, "{PIDFILE}", dir+"/node.pid")
	return []byte(wiredService)
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
