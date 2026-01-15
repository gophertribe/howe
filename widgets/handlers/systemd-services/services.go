package systemdservices

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/fatih/color"

	"github.com/victorgama/howe/helpers"
	"github.com/victorgama/howe/widgets"
)

func handle(ctx context.Context, payload map[string]any, output chan any, wait *sync.WaitGroup) {
	rawServices, ok := payload["services"]
	if !ok {
		output <- fmt.Errorf("systemd-services: services list not declared")
		wait.Done()
		return
	}

	services := []string{}
	if servicesArr, ok := rawServices.([]any); ok {
		for i, c := range servicesArr {
			if n, ok := c.(string); ok {
				services = append(services, n)
			} else {
				output <- fmt.Errorf("systemd-services: item %d in services should be a string", i)
				wait.Done()
				return
			}
		}
	} else {
		output <- fmt.Errorf("systemd-services: services must be a list of strings")
		wait.Done()
		return
	}

	conn, err := dbus.NewWithContext(ctx)
	if err != nil {
		helpers.ReportError(fmt.Sprintf("systemd-services: %s", err))
		output <- fmt.Sprintf("systemd-services: Could not connect", err)
		wait.Done()
		return
	}

	list, err := conn.ListUnitsContext(ctx)
	if err != nil {
		helpers.ReportError(fmt.Sprintf("systemd-services: %s", err))
		output <- fmt.Sprintf("systemd-services: Cannot enumerate units.")
		wait.Done()
		return
	}

	results := [][]string{}

	for _, n := range services {
		r := "not found"
		f := color.FgRed
		for _, s := range list {
			if strings.Replace(strings.ToLower(s.Name), ".service", "", -1) == strings.ToLower(n) {
				f = color.FgWhite
				r = helpers.Titleize(s.SubState)
				switch strings.ToLower(s.SubState) {
				case "running":
					f = color.FgGreen
				case "failed":
					f = color.FgRed
				}
			}
		}
		results = append(results, []string{n, color.New(f).SprintFunc()(r)})
	}

	buf := new(bytes.Buffer)
	w := bufio.NewWriter(buf)
	longest := longestString(results)

	for _, v := range results {
		fmt.Fprintf(w, "    %s    %s\n", padString(v[0], longest), v[1])
	}
	w.Flush()

	output <- "\nServices:\n" + buf.String()
	wait.Done()
}

func init() {
	widgets.Register("systemd-services", handle)
}

func longestString(list [][]string) int {
	longest := 0
	for _, s := range list {
		l := len(s[0])
		if l > longest {
			longest = l
		}
	}

	return longest
}

func padString(str string, size int) string {
	strLen := len(str)
	if strLen >= size {
		return str + ":"
	}
	return str + ":" + strings.Repeat(" ", size-strLen)
}
