package print

import (
	"context"
	"sync"

	"github.com/victorgama/howe/helpers"
	"github.com/victorgama/howe/widgets"
)

var _ widgets.HandlerFunc = handle

func handle(_ context.Context, payload map[string]any, output chan any, wait *sync.WaitGroup) {
	toWrite, err := helpers.TextOrCommand("print", payload)
	if err != nil {
		output <- err
		wait.Done()
		return
	}
	output <- toWrite
	wait.Done()
}

func init() {
	widgets.Register("print", handle)
}
