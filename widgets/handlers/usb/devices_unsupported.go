//go:build !linux

package usb

import (
	"context"
	"sync"

	"github.com/victorgama/howe/widgets"
)

func handle(_ context.Context, _ map[string]any, output chan any, wait *sync.WaitGroup) {
	output <- ""
	wait.Done()
}

func init() {
	widgets.Register("usb-devices", handle)
}
