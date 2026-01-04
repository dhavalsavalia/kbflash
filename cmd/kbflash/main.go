package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dhavalsavalia/kbflash/internal/config"
)

var version = "dev"

func main() {
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.BoolVar(versionFlag, "v", false, "Print version and exit (shorthand)")

	configPath := flag.String("config", "", "Path to config file")
	initConfig := flag.Bool("init", false, "Generate example config file")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("kbflash %s\n", version)
		os.Exit(0)
	}

	if *initConfig {
		path, err := config.GenerateExampleConfig(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created config at %s\n", path)
		os.Exit(0)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// TODO: Launch TUI with config
	fmt.Printf("Loaded config for keyboard: %s\n", cfg.Keyboard.Name)
	fmt.Println("TUI not yet implemented")
}
