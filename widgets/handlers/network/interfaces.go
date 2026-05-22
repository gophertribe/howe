package network

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/fatih/color"

	"github.com/victorgama/howe/helpers"
	"github.com/victorgama/howe/widgets"
)

func handle(_ context.Context, payload map[string]any, output chan any, wait *sync.WaitGroup) {
	// Parse optional include regexes
	var includePatterns []string
	if rawInclude, ok := payload["include"]; ok {
		if arr, ok := rawInclude.([]any); ok {
			for i, item := range arr {
				if s, ok := item.(string); ok {
					includePatterns = append(includePatterns, s)
				} else {
					output <- fmt.Errorf("network-interfaces: item %d in include should be a string", i)
					wait.Done()
					return
				}
			}
		} else {
			output <- fmt.Errorf("network-interfaces: include must be a list of strings")
			wait.Done()
			return
		}
	}

	showIPs := true
	if raw, ok := payload["show_ips"]; ok {
		if b, ok := raw.(bool); ok {
			showIPs = b
		}
	}

	showMAC := false
	if raw, ok := payload["show_mac"]; ok {
		if b, ok := raw.(bool); ok {
			showMAC = b
		}
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		helpers.ReportError(fmt.Sprintf("network-interfaces: %s", err))
		output <- "Network: Could not enumerate interfaces"
		wait.Done()
		return
	}

	results := [][]string{}
	for _, iface := range ifaces {
		// Skip loopback unless explicitly matched
		if iface.Flags&net.FlagLoopback != 0 {
			if !matchesAny(iface.Name, includePatterns) {
				continue
			}
		}

		// Apply include filter if specified
		if len(includePatterns) > 0 && !matchesAny(iface.Name, includePatterns) {
			continue
		}

		state := readOperstate(iface.Name)
		if state == "" {
			if iface.Flags&net.FlagUp != 0 {
				state = "up"
			} else {
				state = "down"
			}
		}

		var parts []string
		parts = append(parts, stateColor(state).SprintFunc()(state))

		if showMAC && iface.HardwareAddr != nil {
			parts = append(parts, iface.HardwareAddr.String())
		}

		if showIPs {
			addrs, err := iface.Addrs()
			if err == nil {
				var ipStrs []string
				for _, addr := range addrs {
					ipStrs = append(ipStrs, addr.String())
				}
				if len(ipStrs) > 0 {
					parts = append(parts, strings.Join(ipStrs, ", "))
				}
			}
		}

		results = append(results, []string{iface.Name, strings.Join(parts, "  ")})
	}

	if len(results) == 0 {
		output <- ""
		wait.Done()
		return
	}

	buf := new(bytes.Buffer)
	w := bufio.NewWriter(buf)
	longest := longestName(results)

	for _, v := range results {
		fmt.Fprintf(w, "    %s    %s\n", padName(v[0], longest), v[1])
	}
	w.Flush()

	output <- "\nNetwork:\n" + buf.String()
	wait.Done()
}

func init() {
	widgets.Register("network-interfaces", handle)
}

func readOperstate(name string) string {
	data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/operstate", name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func matchesAny(name string, patterns []string) bool {
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			continue
		}
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

func stateColor(state string) *color.Color {
	switch strings.ToLower(state) {
	case "up", "unknown":
		return color.New(color.FgGreen)
	case "down":
		return color.New(color.FgRed)
	case "dormant":
		return color.New(color.FgYellow)
	default:
		return color.New(color.FgWhite)
	}
}

func longestName(list [][]string) int {
	longest := 0
	for _, s := range list {
		l := len(s[0])
		if l > longest {
			longest = l
		}
	}
	return longest
}

func padName(str string, size int) string {
	strLen := len(str)
	if strLen >= size {
		return str + ":"
	}
	return str + ":" + strings.Repeat(" ", size-strLen)
}
