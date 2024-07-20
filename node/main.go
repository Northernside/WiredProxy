package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"wirednode/node"

	"wired.rip/wiredutils/config"
	"wired.rip/wiredutils/terminal"
)

//go:embed wirednode.service
var wiredService string

func main() {
	config.Init()
	log.SetFlags(0)
	prefix := fmt.Sprintf("%s.%s » ", config.GetSystemKey(), config.GetWiredHost())
	log.SetPrefix(terminal.PrefixColor + prefix + terminal.Reset)

	if len(os.Args) > 1 {
		args := os.Args[1:]
		switch args[0] {
		case "start":
			node.Run(getFileHash())
		case "install":
			systemdInstall()
		case "setup":
			setup(args[1:])
		case "debug":
			log.Println(runtime.GOARCH)
		}
	} else {
		node.Run(getFileHash())
	}
}

func setup(args []string) {
	if len(args) < 2 {
		log.Fatalln("No arguments provided -> setup <key> <password>")
	}

	key := args[0]
	password := args[1]

	config.Init()
	config.SetSystemKey(key)
	config.SetPassphrase(password)

	log.Println("Setup complete")
}

func systemdInstall() {
	// check if system is linux
	if runtime.GOOS != "linux" {
		log.Fatalln("Systemd installation is only available on linux")
	}

	// check if systemd is available
	cmd := exec.Command("systemctl", "--version")
	err := cmd.Run()
	if err != nil {
		log.Fatalln("Systemd is not available on this system")
	}

	// check if user is root
	if os.Geteuid() != 0 {
		log.Fatalln("You must be root to install the service")
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
