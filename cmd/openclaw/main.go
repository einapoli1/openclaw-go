package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("openclaw-go — lightweight AI agent gateway")
		fmt.Println("Usage: openclaw-go <command>")
		fmt.Println("Commands: gateway, agent, status")
		os.Exit(0)
	}

	switch os.Args[1] {
	case "gateway":
		fmt.Println("Gateway not yet implemented")
	case "agent":
		fmt.Println("Agent not yet implemented")
	case "status":
		fmt.Println("Status not yet implemented")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
