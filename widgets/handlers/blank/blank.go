package blank

import (
	"context"
	"sync"

	"github.com/victorgama/howe/widgets"
)

var _ widgets.HandlerFunc = handle

func handle(_ context.Context, _ map[string]any, output chan any, wait *sync.WaitGroup) {
	output <- " "
	wait.Done()
}

func init() {
	widgets.Register("blank", handle)
}
