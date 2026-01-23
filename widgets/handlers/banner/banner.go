package banner

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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

// textSegment represents a segment of text with its associated color
type textSegment struct {
	text  string
	color string
}

var colorMarkerRegex = regexp.MustCompile(`\[([a-z]+|reset)\]`)

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

// parseColorMarkers parses text with color markers like [red]text[green]more[reset]
// Returns a slice of text segments with their associated colors
func parseColorMarkers(text string) []textSegment {
	var segments []textSegment
	currentColor := ""

	// Find all color markers
	matches := colorMarkerRegex.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		// No color markers found, return entire text with default color
		if text != "" {
			segments = append(segments, textSegment{text: text, color: ""})
		}
		return segments
	}

	lastIndex := 0
	for _, match := range matches {
		// Add text before this marker
		if match[0] > lastIndex {
			textBefore := text[lastIndex:match[0]]
			if textBefore != "" {
				segments = append(segments, textSegment{text: textBefore, color: currentColor})
			}
		}

		// Extract color name from marker
		colorName := text[match[2]:match[3]]
		if colorName == "reset" {
			currentColor = ""
		} else {
			// Validate color exists
			if _, ok := availableColors[colorName]; ok {
				currentColor = colorName
			}
		}

		lastIndex = match[1]
	}

	// Add remaining text after last marker
	if lastIndex < len(text) {
		remainingText := text[lastIndex:]
		if remainingText != "" {
			segments = append(segments, textSegment{text: remainingText, color: currentColor})
		}
	}

	return segments
}

// applyInlineColors processes text with inline color markers and applies colors to figlet output
func applyInlineColors(text string, font *figletlib.Font, defaultColor string) (string, error) {
	segments := parseColorMarkers(text)

	// If no color markers found, use default behavior
	if len(segments) == 1 && segments[0].color == "" {
		figletOutput := figletlib.SprintMsg(text, font, 80, font.Settings(), "left")
		return colorizeOutput(strings.TrimSuffix(figletOutput, "\n"), defaultColor), nil
	}

	// Generate figlet for each segment and store with segment info
	type segmentData struct {
		seg    textSegment
		output []string
	}
	var segmentDataList []segmentData
	for _, seg := range segments {
		if seg.text == "" {
			continue
		}
		figletOutput := figletlib.SprintMsg(seg.text, font, 80, font.Settings(), "left")
		lines := strings.Split(strings.TrimSuffix(figletOutput, "\n"), "\n")
		segmentDataList = append(segmentDataList, segmentData{seg: seg, output: lines})
	}

	if len(segmentDataList) == 0 {
		return "", nil
	}

	// Find maximum number of lines
	maxLines := 0
	for _, data := range segmentDataList {
		if len(data.output) > maxLines {
			maxLines = len(data.output)
		}
	}

	// Combine segments horizontally, applying colors
	var resultLines []string
	for lineIdx := 0; lineIdx < maxLines; lineIdx++ {
		var lineParts []string
		for _, data := range segmentDataList {
			if lineIdx < len(data.output) {
				lineText := data.output[lineIdx]
				// Apply color to this segment's line
				if data.seg.color != "" {
					lineText = colorizeOutput(lineText, data.seg.color)
				} else if defaultColor != "" {
					lineText = colorizeOutput(lineText, defaultColor)
				}
				lineParts = append(lineParts, lineText)
			} else {
				// Pad with spaces if this segment has fewer lines
				if len(data.output) > 0 {
					lineParts = append(lineParts, strings.Repeat(" ", len(data.output[0])))
				}
			}
		}
		resultLines = append(resultLines, strings.Join(lineParts, ""))
	}

	return strings.Join(resultLines, "\n"), nil
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

	// Check if text contains inline color markers
	if colorMarkerRegex.MatchString(toWrite) {
		result, err := applyInlineColors(toWrite, font, fontColor)
		if err != nil {
			output <- err
			wait.Done()
			return
		}
		output <- result
	} else {
		// Use default color behavior
		output <- colorizeOutput(strings.TrimSuffix(figletlib.SprintMsg(toWrite, font, 80, font.Settings(), "left"), "\n"), fontColor)
	}
	wait.Done()
}

// ListAvailableFonts returns a list of all available embedded font names (without .flf extension)
func ListAvailableFonts() ([]string, error) {
	entries, err := embeddedFonts.ReadDir("fonts")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded fonts: %w", err)
	}

	fonts := make([]string, 0, len(entries))
	for _, entry := range entries {
		// entry is of type fs.DirEntry
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".flf") {
			fontName := strings.TrimSuffix(entry.Name(), ".flf")
			fonts = append(fonts, fontName)
		}
	}

	// Reference fs package to satisfy compiler
	_ = fs.FileMode(0)

	sort.Strings(fonts)
	return fonts, nil
}

// PreviewFont generates a preview of the given font with the specified text
func PreviewFont(fontName, text string) (string, error) {
	font, err := loadFiglet(fontName)
	if err != nil {
		return "", fmt.Errorf("failed to load font '%s': %w", fontName, err)
	}

	preview := figletlib.SprintMsg(text, font, 80, font.Settings(), "left")
	return strings.TrimSuffix(preview, "\n"), nil
}

func init() {
	widgets.Register("banner", handle)
}
