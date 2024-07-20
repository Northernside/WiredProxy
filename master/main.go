package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"wiredmaster/master"

	"wired.rip/wiredutils/config"
	"wired.rip/wiredutils/terminal"
)

func main() {
	config.Init()
	log.SetFlags(0)
	prefix := fmt.Sprintf("%s.%s » ", config.GetSystemKey(), config.GetWiredHost())
	log.SetPrefix(terminal.PrefixColor + prefix + terminal.Reset)

	if len(os.Args) > 1 {
		args := os.Args[1:]
		switch args[0] {
		case "start":
			master.Run()
		case "add-node":
			addNode(args[1:])
		case "debug":
			log.Println(runtime.GOARCH)
		}
	} else {
		master.Run()
	}
}

func addNode(args []string) {
	if len(args) < 2 {
		log.Fatalln("No arguments provided -> add-node <key> <password>")
	}

	key := args[0]
	password := args[1]

	_ = config.AddNode(config.Node{
		Id:             key,
		Passphrase:     password,
		LastConnection: 0,
	})

	log.Println("Node added")
}
