/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package main

import (
	"fmt"
	"os"
	
	"github.com/kajvans/foundry/cmd"
	"github.com/kajvans/foundry/internal/config"
	"github.com/kajvans/foundry/internal/detect"
)

func main() {

	//check if config exists
	var ConfigExists bool = ensureConfigExists()

	if ConfigExists {
		// Proceed with application logic
		//fmt.Println("Configuration loaded successfully.")
	} else {
		//run detect
		//fmt.Println("Running configuration detection...")
		detect.ScanSystem()

		fmt.Println("You can now run 'foundry config' to view or update your configuration.")
	}

	cmd.Execute()
}

func ensureConfigExists() bool {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		return false
	}
	//if not run config initialization
	if cfg == nil {
		config.InitConfig()
	}
	return true
}