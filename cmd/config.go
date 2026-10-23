package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/zengjie/try/core"
)

// RunConfig implements `try config [init]`: with no subcommand it shows the
// effective ranking configuration and where it comes from; `init` scaffolds
// the config file with the defaults for editing.
func RunConfig(sub string) error {
	switch sub {
	case "":
		return showConfig()
	case "init":
		return initConfig()
	default:
		return fmt.Errorf("unknown config subcommand %q (use 'try config' or 'try config init')", sub)
	}
}

func showConfig() error {
	path := core.GetConfigPath()

	source := "using built-in defaults"
	if _, err := os.Stat(path); err == nil {
		source = "loaded"
	}
	fmt.Printf("Config file: %s (%s)\n\n", path, source)

	data, err := json.MarshalIndent(core.LoadConfig(), "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))

	if source != "loaded" {
		fmt.Println("\nRun 'try config init' to create the file, then edit it to tune ranking.")
	}
	return nil
}

func initConfig() error {
	path, err := core.WriteDefaultConfigFile()
	if err != nil {
		return err
	}
	fmt.Printf("Created %s\n", path)
	fmt.Println("Edit it to tune ranking; missing or invalid fields fall back to the defaults.")
	return nil
}
