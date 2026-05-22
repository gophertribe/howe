//go:build linux

package usb

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/fatih/color"

	"github.com/victorgama/howe/helpers"
	"github.com/victorgama/howe/widgets"
)

type usbDevice struct {
	name         string
	manufacturer string
	product      string
	vendorID     string
	productID    string
	busPort      string
}

func handle(_ context.Context, payload map[string]any, output chan any, wait *sync.WaitGroup) {
	var vendorIDFilter, productIDFilter string
	var vendorNameRe, productNameRe *regexp.Regexp

	if raw, ok := payload["vendor_id"]; ok {
		if s, ok := raw.(string); ok {
			vendorIDFilter = strings.ToLower(s)
		}
	}
	if raw, ok := payload["product_id"]; ok {
		if s, ok := raw.(string); ok {
			productIDFilter = strings.ToLower(s)
		}
	}
	if raw, ok := payload["vendor_name"]; ok {
		if s, ok := raw.(string); ok {
			var err error
			vendorNameRe, err = regexp.Compile(s)
			if err != nil {
				output <- fmt.Errorf("usb-devices: invalid vendor_name regex: %w", err)
				wait.Done()
				return
			}
		}
	}
	if raw, ok := payload["product_name"]; ok {
		if s, ok := raw.(string); ok {
			var err error
			productNameRe, err = regexp.Compile(s)
			if err != nil {
				output <- fmt.Errorf("usb-devices: invalid product_name regex: %w", err)
				wait.Done()
				return
			}
		}
	}

	devices, err := enumerateUSBDevices()
	if err != nil {
		helpers.ReportError(fmt.Sprintf("usb-devices: %s", err))
		output <- "USB: Could not enumerate devices"
		wait.Done()
		return
	}

	results := [][]string{}
	for _, dev := range devices {
		if vendorIDFilter != "" && strings.ToLower(dev.vendorID) != vendorIDFilter {
			continue
		}
		if productIDFilter != "" && strings.ToLower(dev.productID) != productIDFilter {
			continue
		}
		if vendorNameRe != nil && !vendorNameRe.MatchString(dev.manufacturer) {
			continue
		}
		if productNameRe != nil && !productNameRe.MatchString(dev.product) {
			continue
		}

		display := dev.name
		if display == "" {
			display = "Unknown Device"
		}
		display = fmt.Sprintf("%s (%s:%s)", display, dev.vendorID, dev.productID)
		results = append(results, []string{display, fmt.Sprintf("@ %s", dev.busPort)})
	}

	if len(results) == 0 {
		output <- ""
		wait.Done()
		return
	}

	buf := new(bytes.Buffer)
	w := bufio.NewWriter(buf)
	for _, v := range results {
		fmt.Fprintf(w, "    %s  %s\n", v[0], color.New(color.FgHiBlack).SprintFunc()(v[1]))
	}
	w.Flush()

	output <- "\nUSB:\n" + buf.String()
	wait.Done()
}

func init() {
	widgets.Register("usb-devices", handle)
}

func enumerateUSBDevices() ([]usbDevice, error) {
	basePath := "/sys/bus/usb/devices"
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil, err
	}

	var devices []usbDevice
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Skip interface entries (e.g., 1-1:1.0); keep only device entries
		if strings.Contains(name, ":") {
			continue
		}

		devPath := filepath.Join(basePath, name)
		vendorBytes, err := os.ReadFile(filepath.Join(devPath, "idVendor"))
		if err != nil {
			continue
		}
		productBytes, err := os.ReadFile(filepath.Join(devPath, "idProduct"))
		if err != nil {
			continue
		}

		vendorID := strings.TrimSpace(string(vendorBytes))
		productID := strings.TrimSpace(string(productBytes))

		manufacturer := ""
		if b, err := os.ReadFile(filepath.Join(devPath, "manufacturer")); err == nil {
			manufacturer = strings.TrimSpace(string(b))
		}
		product := ""
		if b, err := os.ReadFile(filepath.Join(devPath, "product")); err == nil {
			product = strings.TrimSpace(string(b))
		}

		displayName := strings.TrimSpace(manufacturer + " " + product)
		if displayName == "" {
			displayName = fmt.Sprintf("USB Device %s:%s", vendorID, productID)
		}

		devices = append(devices, usbDevice{
			name:         displayName,
			manufacturer: manufacturer,
			product:      product,
			vendorID:     vendorID,
			productID:    productID,
			busPort:      name,
		})
	}

	return devices, nil
}
