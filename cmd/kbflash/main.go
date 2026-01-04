package main

import (
	"flag"
	"fmt"
	"os"
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
		fmt.Println("TODO: Generate example config")
		os.Exit(0)
	}

	// TODO: Load config and launch TUI
	_ = configPath
	fmt.Println("kbflash - Hackable keyboard firmware flasher")
	fmt.Println("Run with --help for usage")
}
