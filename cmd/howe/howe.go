package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/victorgama/howe/widgets"
	_ "github.com/victorgama/howe/widgets/handlers/banner"
	_ "github.com/victorgama/howe/widgets/handlers/blank"
	_ "github.com/victorgama/howe/widgets/handlers/disks"
	_ "github.com/victorgama/howe/widgets/handlers/docker"
	_ "github.com/victorgama/howe/widgets/handlers/load"
	_ "github.com/victorgama/howe/widgets/handlers/print"
	_ "github.com/victorgama/howe/widgets/handlers/systemd-services"
	_ "github.com/victorgama/howe/widgets/handlers/updates"
	_ "github.com/victorgama/howe/widgets/handlers/uptime"

	"github.com/victorgama/howe/config"
)

var (
	configPath string
	noColor    bool
)

func main() {
	// Set color output based on flag before executing commands
	color.NoColor = noColor

	err := rootCmd.Execute()
	if err != nil {
		// Colorize error output in red
		red := color.New(color.FgRed)
		red.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "howe",
	Short: "A modern MOTD replacement",
	Long: `Howe provides a replacement for MOTD. Its contents can be customized
in order to provide relevant information about your system.

Howe contains several widgets that collect and process system information
every time the utility is executed. Widgets can be configured through a
configuration file (default: /etc/howe/config.yml).

If no command is specified, 'run' is executed by default.`,
	Version:       version,
	RunE:          runHowe,
	SilenceUsage:  true, // Don't show usage on error, we'll handle it ourselves
	SilenceErrors: true, // Don't show errors, we'll handle them with color ourselves
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "/etc/howe/config.yml", "path to configuration file")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
}

func runHowe(cmd *cobra.Command, args []string) error {
	// Set color output based on flag
	color.NoColor = noColor

	// Load and validate config
	cfg, err := loadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Execute widgets
	ctx := context.Background()
	results, err := executeWidgets(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to execute widgets: %w", err)
	}

	// Output results
	for _, result := range results {
		if err, ok := result.(error); ok {
			return fmt.Errorf("widget error: %w", err)
		}
		if str, ok := result.(string); ok {
			fmt.Println(strings.Trim(str, "\n"))
		}
	}

	return nil
}

func loadConfig(path string) (*config.Root, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s not found. Please refer to the documentation", path)
		}
		return nil, fmt.Errorf("error accessing config file: %w", err)
	}

	configData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg config.Root
	err = yaml.Unmarshal(configData, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &cfg, nil
}

func executeWidgets(ctx context.Context, cfg *config.Root) ([]any, error) {
	var wg sync.WaitGroup
	results := []chan any{}

	for i, w := range cfg.Items {
		rawName, ok := w["type"]
		if !ok {
			return nil, fmt.Errorf("widget %d is missing a type attribute", i+1)
		}

		name, ok := rawName.(string)
		if !ok {
			return nil, fmt.Errorf("widget %d has an invalid type attribute", i+1)
		}

		handler, ok := widgets.Handlers[name]
		if !ok {
			return nil, fmt.Errorf("widget %d uses unknown type %s", i+1, name)
		}

		wg.Add(1)
		output := make(chan any, 1)
		results = append(results, output)
		go handler(ctx, w, output, &wg)
	}

	wg.Wait()

	outputs := make([]any, 0, len(results))
	for _, ch := range results {
		outputs = append(outputs, <-ch)
	}

	return outputs, nil
}
