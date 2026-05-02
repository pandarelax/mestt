package audio

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
)

func buildRecordArgs(goos string, opts RecordOptions, outputPath string) ([]string, error) {
	sampleRate := opts.SampleRate
	if sampleRate <= 0 {
		sampleRate = 16000
	}

	switch goos {
	case "linux":
		driver := opts.Driver
		if driver == "" {
			driver = "pulse"
		}
		device := opts.Device
		if device == "" || device == "default" {
			device = "default"
		}
		return []string{
			"-hide_banner",
			"-loglevel", "error",
			"-y",
			"-f", driver,
			"-i", device,
			"-ar", fmt.Sprintf("%d", sampleRate),
			"-ac", "1",
			"-c:a", "pcm_s16le",
			outputPath,
		}, nil
	case "darwin":
		device := opts.Device
		if device == "" || device == "default" {
			device = ":0"
		} else if !strings.Contains(device, ":") {
			device = ":" + device
		}
		return []string{
			"-hide_banner",
			"-loglevel", "error",
			"-y",
			"-f", "avfoundation",
			"-i", device,
			"-ar", fmt.Sprintf("%d", sampleRate),
			"-ac", "1",
			"-c:a", "pcm_s16le",
			outputPath,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported operating system %q", goos)
	}
}

func buildListDevicesArgs(goos, driver string) ([]string, error) {
	switch goos {
	case "linux":
		if driver == "" {
			driver = "pulse"
		}
		return []string{"-hide_banner", "-sources", driver}, nil
	case "darwin":
		return []string{"-hide_banner", "-f", "avfoundation", "-list_devices", "true", "-i", ""}, nil
	default:
		return nil, fmt.Errorf("unsupported operating system %q", goos)
	}
}

func parseDevices(goos, output string) []Device {
	if goos == "darwin" {
		return parseDarwinDevices(output)
	}
	return parseLinuxDevices(output)
}

func currentOS() string {
	return runtime.GOOS
}

func parseDarwinDevices(output string) []Device {
	var devices []Device
	audioSection := false
	re := regexp.MustCompile(`\[(\d+)\]\s+(.+)$`)
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "AVFoundation audio devices") {
			audioSection = true
			continue
		}
		if strings.Contains(trimmed, "AVFoundation video devices") {
			audioSection = false
			continue
		}
		if !audioSection {
			continue
		}
		match := re.FindStringSubmatch(trimmed)
		if len(match) != 3 {
			continue
		}
		devices = append(devices, Device{ID: match[1], Name: match[2], Driver: "avfoundation", Default: match[1] == "0"})
	}
	return devices
}

func parseLinuxDevices(output string) []Device {
	var devices []Device
	re := regexp.MustCompile(`^\s*(\*)?\s*([^\s]+)\s+\[(.+)\]\s*$`)
	for _, line := range strings.Split(output, "\n") {
		match := re.FindStringSubmatch(line)
		if len(match) != 4 {
			continue
		}
		devices = append(devices, Device{ID: match[2], Name: match[3], Driver: "pulse", Default: match[1] == "*" || match[2] == "default"})
	}
	return devices
}
