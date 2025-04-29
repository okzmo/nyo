package main

import (
	"fmt"
	"os"

	"github.com/okzmo/nyo/src/commands"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: nyo <command> [arguments]")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "deploy":
		err := commands.Deploy()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}
