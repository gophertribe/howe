package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/victorgama/howe/widgets/handlers/banner"
)

// fontsCmd represents the fonts command
var fontsCmd = &cobra.Command{
	Use:   "fonts",
	Short: "List available fonts",
	Long: `List all available embedded fonts that can be used with the banner widget.
Optionally preview a specific font with sample text.`,
	RunE: listFonts,
}

var (
	fontPreview     string
	fontPreviewText string
)

func init() {
	rootCmd.AddCommand(fontsCmd)
	fontsCmd.Flags().StringVarP(&fontPreview, "preview", "p", "", "preview a specific font by name")
	fontsCmd.Flags().StringVarP(&fontPreviewText, "text", "t", "Howe", "text to use for font preview")
}

func listFonts(cmd *cobra.Command, args []string) error {
	// If preview flag is set, show preview of that font
	if fontPreview != "" {
		return previewFont(fontPreview, fontPreviewText)
	}

	// Otherwise, list all available fonts
	fonts, err := banner.ListAvailableFonts()
	if err != nil {
		return fmt.Errorf("failed to list fonts: %w", err)
	}

	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Fprintf(os.Stderr, "Available fonts (%d):\n\n", len(fonts))

	// Display fonts in columns for better readability
	cols := 3
	for i := 0; i < len(fonts); i += cols {
		line := make([]string, 0, cols)
		for j := 0; j < cols && i+j < len(fonts); j++ {
			line = append(line, cyan(fonts[i+j]))
		}
		fmt.Println(strings.Join(line, "  "))
	}

	fmt.Fprintf(os.Stderr, "\nUse --preview <font-name> to preview a specific font.\n")
	fmt.Fprintf(os.Stderr, "Example: howe fonts --preview doom\n")

	return nil
}

func previewFont(fontName, text string) error {
	preview, err := banner.PreviewFont(fontName, text)
	if err != nil {
		return err
	}

	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Fprintf(os.Stderr, "Font: %s\n", cyan(fontName))
	fmt.Fprintf(os.Stderr, "Text: %s\n\n", text)
	fmt.Println(preview)
	fmt.Fprintf(os.Stderr, "\nUse in config: font: %s\n", fontName)

	return nil
}
