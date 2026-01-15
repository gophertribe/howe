package banner

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/arsham/rainbow/rainbow"
	"github.com/fatih/color"
	"github.com/lukesampson/figlet/figletlib"
	"github.com/victorgama/howe/helpers"
	"github.com/victorgama/howe/widgets"
)

//go:embed fonts/*.flf
var embeddedFonts embed.FS

const defaultFigletsDir = "/usr/share/howe"

var availableColors = map[string]color.Attribute{
	"red":     color.FgRed,
	"green":   color.FgGreen,
	"yellow":  color.FgYellow,
	"blue":    color.FgBlue,
	"magenta": color.FgMagenta,
	"cyan":    color.FgCyan,
	"white":   color.FgWhite,
	// Value of this next key isn't really used, but the key must be present.
	"rainbow": color.FgWhite,
}

func loadFiglet(from string) (*figletlib.Font, error) {
	var buf []byte
	var err error

	// If no font specified, use standard.flf as default
	if from == "" {
		from = "standard"
	}

	// Ensure .flf extension
	fontName := from
	if filepath.Ext(from) == "" {
		fontName = from + ".flf"
	}

	// First, try to load from embedded fonts
	embeddedPath := "fonts/" + fontName
	buf, err = embeddedFonts.ReadFile(embeddedPath)
	if err == nil {
		return figletlib.ReadFontFromBytes(buf)
	}

	// If not found in embedded fonts, try file system
	// If relative path, check default directory first
	if !filepath.IsAbs(from) {
		fsPath := filepath.Join(defaultFigletsDir, fontName)
		buf, err = os.ReadFile(fsPath)
		if err == nil {
			return figletlib.ReadFontFromBytes(buf)
		}
	}

	// Try absolute path if provided
	if filepath.IsAbs(from) {
		buf, err = os.ReadFile(from)
		if err == nil {
			return figletlib.ReadFontFromBytes(buf)
		}
	}

	// If all else fails, fall back to standard.flf from embedded fonts
	if from != "standard" {
		helpers.ReportError(fmt.Sprintf("banner: font '%s' not found, falling back to standard font", from))
		buf, err = embeddedFonts.ReadFile("fonts/standard.flf")
		if err != nil {
			return nil, fmt.Errorf("failed to load standard font: %w", err)
		}
		return figletlib.ReadFontFromBytes(buf)
	}

	return nil, fmt.Errorf("failed to load font '%s': %w", from, err)
}

func colorizeOutput(value, colorName string) string {
	if colorName == "rainbow" {
		buffer := []byte{}
		output := bytes.NewBuffer(buffer)
		r := rainbow.Light{
			Reader: strings.NewReader(value),
			Writer: output,
		}
		err := r.Paint()
		if err != nil {
			return ""
		}
		return output.String()
	}
	foreground := color.New(availableColors[colorName]).SprintFunc()
	return foreground(value)
}

var _ widgets.HandlerFunc = handle

func handle(_ context.Context, payload map[string]any, output chan any, wait *sync.WaitGroup) {
	toWrite, err := helpers.TextOrCommand("banner", payload)
	if err != nil {
		output <- err
		wait.Done()
		return
	}

	fontNameOrPath := ""
	if font, ok := payload["font"]; ok {
		if strFont, ok := font.(string); ok {
			fontNameOrPath = strFont
		} else {
			output <- fmt.Errorf("font property must be a string")
			wait.Done()
			return
		}
	}

	fontColor := "magenta"
	if color, ok := payload["color"]; ok {
		if strColor, ok := color.(string); ok {
			fontColor = strings.ToLower(strColor)
		} else {
			output <- fmt.Errorf("color property must be a string")
			wait.Done()
			return
		}
	}

	_, valid := availableColors[fontColor]

	if !valid {
		colors := make([]string, len(availableColors))
		for k := range availableColors {
			colors = append(colors, k)
		}
		output <- fmt.Errorf("invalid color; valid values are %s", strings.Join(colors, ", "))
		wait.Done()
		return
	}

	font, err := loadFiglet(fontNameOrPath)
	if err != nil {
		output <- err
		wait.Done()
		return
	}

	output <- colorizeOutput(strings.TrimSuffix(figletlib.SprintMsg(toWrite, font, 80, font.Settings(), "left"), "\n"), fontColor)
	wait.Done()
}

func init() {
	widgets.Register("banner", handle)
}
