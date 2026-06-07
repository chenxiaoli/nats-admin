package main

import (
	"fmt"
	"os"
)

func main() {
	cmd := "server"
	args := os.Args[1:]
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}

	var err error
	switch cmd {
	case "server":
		err = runServer()
	case "migrate":
		err = runMigrate(args)
	case "bootstrap":
		err = runBootstrap()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\nusage: nats-admin [server|migrate|bootstrap]\n", cmd)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", cmd, err)
		os.Exit(1)
	}
}
