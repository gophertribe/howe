package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"

	"github.com/victorgama/howe/config"
	"github.com/victorgama/howe/widgets"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Validate and display configuration",
	Long: `Validate the configuration file and display any errors.
If validation passes, the configuration will be displayed in a formatted way.`,
	RunE: validateConfig,
}

var configShow bool

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.Flags().BoolVar(&configShow, "show", false, "display the parsed configuration")
}

func validateConfig(cmd *cobra.Command, args []string) error {
	var cfg *config.Root
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	// Validate widgets
	for i, item := range cfg.Items {
		rawName, ok := item["type"]
		if !ok {
			return fmt.Errorf("widget %d is missing a type attribute", i+1)
		}

		name, ok := rawName.(string)
		if !ok {
			return fmt.Errorf("widget %d has an invalid type attribute", i+1)
		}

		_, ok = widgets.Handlers[name]
		if !ok {
			return fmt.Errorf("widget %d uses unknown type %s", i+1, name)
		}
	}

	fmt.Fprintf(os.Stderr, "✓ Configuration file is valid: %s\n", configPath)
	fmt.Fprintf(os.Stderr, "  Found %d widget(s)\n", len(cfg.Items))

	if configShow {
		fmt.Println("\nParsed configuration:")
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		fmt.Println(string(data))
	}

	return nil
}
