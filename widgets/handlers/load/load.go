package load

import (
	"context"
	"fmt"
	"sync"

	sigar "github.com/cloudfoundry/gosigar"

	"github.com/victorgama/howe/helpers"
	"github.com/victorgama/howe/widgets"
)

var _ widgets.HandlerFunc = handle

func handle(_ context.Context, payload map[string]any, output chan any, wait *sync.WaitGroup) {
	avg := sigar.LoadAverage{}
	err := avg.Get()
	if err != nil {
		helpers.ReportError(fmt.Sprintf("load: %s", err))
		output <- "No load information available"
		wait.Done()
		return
	}
	output <- fmt.Sprintf("load average: %.2f, %.2f, %.2f", avg.One, avg.Five, avg.Fifteen)
	wait.Done()
}

func init() {
	widgets.Register("load", handle)
}
