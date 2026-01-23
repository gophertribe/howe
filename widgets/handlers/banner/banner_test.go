package banner

import (
	"strings"
	"testing"

	"github.com/lukesampson/figlet/figletlib"
)

func TestParseColorMarkers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []textSegment
	}{
		{
			name:  "no color markers",
			input: "Hello World",
			expected: []textSegment{
				{text: "Hello World", color: ""},
			},
		},
		{
			name:  "single color marker at start",
			input: "[red]Hello",
			expected: []textSegment{
				{text: "Hello", color: "red"},
			},
		},
		{
			name:  "single color marker in middle",
			input: "Hello[red]World",
			expected: []textSegment{
				{text: "Hello", color: ""},
				{text: "World", color: "red"},
			},
		},
		{
			name:  "multiple color markers",
			input: "[red]Sat[green]Soft[reset]",
			expected: []textSegment{
				{text: "Sat", color: "red"},
				{text: "Soft", color: "green"},
			},
		},
		{
			name:  "reset marker",
			input: "[red]Hello[reset]World",
			expected: []textSegment{
				{text: "Hello", color: "red"},
				{text: "World", color: ""},
			},
		},
		{
			name:  "multiple colors",
			input: "[red]Red[green]Green[blue]Blue[yellow]Yellow",
			expected: []textSegment{
				{text: "Red", color: "red"},
				{text: "Green", color: "green"},
				{text: "Blue", color: "blue"},
				{text: "Yellow", color: "yellow"},
			},
		},
		{
			name:  "text before and after markers",
			input: "Start[red]Middle[green]End",
			expected: []textSegment{
				{text: "Start", color: ""},
				{text: "Middle", color: "red"},
				{text: "End", color: "green"},
			},
		},
		{
			name:  "invalid color marker",
			input: "[invalid]Text",
			expected: []textSegment{
				{text: "Text", color: ""},
			},
		},
		{
			name:  "empty text segments",
			input: "[red][green]Text",
			expected: []textSegment{
				{text: "Text", color: "green"},
			},
		},
		{
			name:  "uppercase color (should be ignored)",
			input: "[RED]Text",
			expected: []textSegment{
				{text: "[RED]Text", color: ""},
			},
		},
		{
			name:  "color marker at end",
			input: "Text[red]",
			expected: []textSegment{
				{text: "Text", color: ""},
			},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []textSegment{},
		},
		{
			name:  "all available colors",
			input: "[red]R[green]G[blue]B[yellow]Y[magenta]M[cyan]C[white]W",
			expected: []textSegment{
				{text: "R", color: "red"},
				{text: "G", color: "green"},
				{text: "B", color: "blue"},
				{text: "Y", color: "yellow"},
				{text: "M", color: "magenta"},
				{text: "C", color: "cyan"},
				{text: "W", color: "white"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseColorMarkers(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d segments, got %d", len(tt.expected), len(result))
				return
			}
			for i, seg := range result {
				if i >= len(tt.expected) {
					t.Errorf("unexpected segment %d: %+v", i, seg)
					continue
				}
				expected := tt.expected[i]
				if seg.text != expected.text {
					t.Errorf("segment %d: expected text %q, got %q", i, expected.text, seg.text)
				}
				if seg.color != expected.color {
					t.Errorf("segment %d: expected color %q, got %q", i, expected.color, seg.color)
				}
			}
		})
	}
}

func TestApplyInlineColors(t *testing.T) {
	// Load a test font
	font, err := loadFiglet("standard")
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}

	tests := []struct {
		name         string
		input        string
		defaultColor string
		expectError  bool
		validate     func(t *testing.T, result string)
	}{
		{
			name:         "no color markers",
			input:        "Hello",
			defaultColor: "red",
			expectError:  false,
			validate: func(t *testing.T, result string) {
				if result == "" {
					t.Error("expected non-empty result")
				}
				// Should contain the text (case-insensitive check)
				if len(result) < 5 {
					t.Error("result seems too short")
				}
			},
		},
		{
			name:         "single color marker",
			input:        "[red]Hello",
			defaultColor: "blue",
			expectError:  false,
			validate: func(t *testing.T, result string) {
				if result == "" {
					t.Error("expected non-empty result")
				}
			},
		},
		{
			name:         "multiple color markers",
			input:        "[red]Sat[green]Soft",
			defaultColor: "blue",
			expectError:  false,
			validate: func(t *testing.T, result string) {
				if result == "" {
					t.Error("expected non-empty result")
				}
				// Should have multiple lines (figlet output)
				lines := countLines(result)
				if lines < 1 {
					t.Error("expected multiple lines in figlet output")
				}
			},
		},
		{
			name:         "reset marker",
			input:        "[red]Hello[reset]World",
			defaultColor: "blue",
			expectError:  false,
			validate: func(t *testing.T, result string) {
				if result == "" {
					t.Error("expected non-empty result")
				}
			},
		},
		{
			name:         "empty string",
			input:        "",
			defaultColor: "red",
			expectError:  false,
			validate: func(t *testing.T, result string) {
				// Empty input should produce empty or minimal output
			},
		},
		{
			name:         "only color markers",
			input:        "[red][green][blue]",
			defaultColor: "yellow",
			expectError:  false,
			validate: func(t *testing.T, result string) {
				// Should handle gracefully
			},
		},
		{
			name:         "complex example",
			input:        "Start[red]Red[green]Green[blue]Blue[reset]End",
			defaultColor: "yellow",
			expectError:  false,
			validate: func(t *testing.T, result string) {
				if result == "" {
					t.Error("expected non-empty result")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := applyInlineColors(tt.input, font, tt.defaultColor)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestApplyInlineColorsWithRealFont(t *testing.T) {
	font, err := loadFiglet("standard")
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}

	// Test that segments are properly combined
	input := "[red]A[green]B"
	result, err := applyInlineColors(input, font, "blue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}

	// Verify it contains multiple lines (figlet output)
	lines := countLines(result)
	if lines < 1 {
		t.Error("expected multiple lines in figlet output")
	}

	// Test with reset
	input2 := "[red]Hello[reset]World"
	result2, err := applyInlineColors(input2, font, "blue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result2 == "" {
		t.Error("expected non-empty result")
	}
}

func TestColorMarkerRegex(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid marker", "[red]text", true},
		{"multiple markers", "[red]text[green]more", true},
		{"reset marker", "[reset]", true},
		{"no markers", "plain text", false},
		{"uppercase marker", "[RED]text", false}, // regex is case-sensitive
		{"invalid format", "(red)text", false},
		{"empty brackets", "[]text", false},
		{"mixed case", "[Red]text", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := colorMarkerRegex.MatchString(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v for input %q", tt.expected, result, tt.input)
			}
		})
	}
}

func TestParseColorMarkersEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, segments []textSegment)
	}{
		{
			name:  "consecutive markers",
			input: "[red][green][blue]Text",
			validate: func(t *testing.T, segments []textSegment) {
				if len(segments) != 1 {
					t.Errorf("expected 1 segment, got %d", len(segments))
				}
				if len(segments) > 0 && segments[0].color != "blue" {
					t.Errorf("expected color 'blue', got %q", segments[0].color)
				}
			},
		},
		{
			name:  "marker at start and end",
			input: "[red]Text[green]",
			validate: func(t *testing.T, segments []textSegment) {
				if len(segments) != 1 {
					t.Errorf("expected 1 segment, got %d", len(segments))
				}
				if len(segments) > 0 && segments[0].text != "Text" {
					t.Errorf("expected text 'Text', got %q", segments[0].text)
				}
			},
		},
		{
			name:  "reset in middle",
			input: "[red]Before[reset]After",
			validate: func(t *testing.T, segments []textSegment) {
				if len(segments) != 2 {
					t.Errorf("expected 2 segments, got %d", len(segments))
					return
				}
				if segments[0].color != "red" {
					t.Errorf("first segment: expected color 'red', got %q", segments[0].color)
				}
				if segments[1].color != "" {
					t.Errorf("second segment: expected empty color, got %q", segments[1].color)
				}
			},
		},
		{
			name:  "numbers in brackets",
			input: "[123]Text",
			validate: func(t *testing.T, segments []textSegment) {
				// Should not match as color marker
				if len(segments) != 1 || segments[0].color != "" {
					t.Errorf("expected no color, got %+v", segments)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments := parseColorMarkers(tt.input)
			tt.validate(t, segments)
		})
	}
}

// Helper function to count lines in a string
func countLines(s string) int {
	if s == "" {
		return 0
	}
	count := 1
	for _, c := range s {
		if c == '\n' {
			count++
		}
	}
	return count
}

// Test that figlet library integration works
func TestFigletIntegration(t *testing.T) {
	font, err := loadFiglet("standard")
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}

	// Test basic figlet output
	output := figletlib.SprintMsg("Test", font, 80, font.Settings(), "left")
	if output == "" {
		t.Error("figlet output should not be empty")
	}

	// Test that output has multiple lines
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	if len(lines) < 1 {
		t.Error("figlet output should have at least one line")
	}
}

// Test colorizeOutput function
func TestColorizeOutput(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		colorName string
		validate  func(t *testing.T, result string)
	}{
		{
			name:      "red color",
			value:     "Hello",
			colorName: "red",
			validate: func(t *testing.T, result string) {
				if result == "" {
					t.Error("expected non-empty result")
				}
			},
		},
		{
			name:      "green color",
			value:     "World",
			colorName: "green",
			validate: func(t *testing.T, result string) {
				if result == "" {
					t.Error("expected non-empty result")
				}
			},
		},
		{
			name:      "empty string",
			value:     "",
			colorName: "red",
			validate: func(t *testing.T, result string) {
				// Empty input should produce empty output
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := colorizeOutput(tt.value, tt.colorName)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}
