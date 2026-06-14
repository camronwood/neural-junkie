package hardware

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// MemoryUsage holds live system RAM stats.
type MemoryUsage struct {
	TotalBytes     uint64  `json:"total_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	UsedPercent    float64  `json:"used_percent"`
	// Darwin Activity Monitor breakdown (App + Wired + Compressed).
	AppMemoryBytes        uint64 `json:"app_memory_bytes,omitempty"`
	WiredMemoryBytes      uint64 `json:"wired_memory_bytes,omitempty"`
	CompressedMemoryBytes uint64 `json:"compressed_memory_bytes,omitempty"`
}

// MemoryUsageSnapshot probes current system RAM pressure.
func MemoryUsageSnapshot() (MemoryUsage, error) {
	switch runtime.GOOS {
	case "linux":
		return memoryUsageLinux()
	case "darwin":
		return memoryUsageDarwin()
	case "windows":
		return memoryUsageWindows()
	default:
		return MemoryUsage{}, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func memoryUsageFromTotalAvailable(total, available uint64) MemoryUsage {
	if total == 0 {
		return MemoryUsage{}
	}
	if available > total {
		available = total
	}
	used := total - available
	pct := float64(used) / float64(total) * 100
	return MemoryUsage{
		TotalBytes:     total,
		AvailableBytes: available,
		UsedBytes:      used,
		UsedPercent:    pct,
	}
}

func memoryUsageLinux() (MemoryUsage, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return MemoryUsage{}, err
	}
	defer f.Close()
	total, available, err := parseLinuxMeminfo(f)
	if err != nil {
		return MemoryUsage{}, err
	}
	return memoryUsageFromTotalAvailable(total, available), nil
}

func parseLinuxMeminfo(r io.Reader) (total, available uint64, err error) {
	var memTotal, memAvailable uint64
	foundTotal := false
	foundAvailable := false

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			memTotal, err = parseMeminfoKB(line)
			if err != nil {
				return 0, 0, err
			}
			foundTotal = true
		case strings.HasPrefix(line, "MemAvailable:"):
			memAvailable, err = parseMeminfoKB(line)
			if err != nil {
				return 0, 0, err
			}
			foundAvailable = true
		}
		if foundTotal && foundAvailable {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}
	if !foundTotal {
		return 0, 0, fmt.Errorf("MemTotal not found in /proc/meminfo")
	}
	if !foundAvailable {
		return 0, 0, fmt.Errorf("MemAvailable not found in /proc/meminfo")
	}
	return memTotal, memAvailable, nil
}

func parseMeminfoKB(line string) (uint64, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, fmt.Errorf("invalid meminfo line: %q", line)
	}
	kb, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0, err
	}
	return kb * 1024, nil
}

func memoryUsageDarwin() (MemoryUsage, error) {
	total, err := TotalMemoryBytes()
	if err != nil {
		return MemoryUsage{}, err
	}

	out, err := exec.Command("vm_stat").Output()
	if err != nil {
		return MemoryUsage{}, fmt.Errorf("vm_stat: %w", err)
	}
	stats, pageSize, err := parseDarwinVmStat(string(out))
	if err != nil {
		return MemoryUsage{}, err
	}

	pageableOut, err := exec.Command("sysctl", "-n", "vm.page_pageable_internal_count").Output()
	if err != nil {
		return MemoryUsage{}, fmt.Errorf("sysctl vm.page_pageable_internal_count: %w", err)
	}
	pageablePages, err := strconv.ParseUint(strings.TrimSpace(string(pageableOut)), 10, 64)
	if err != nil {
		return MemoryUsage{}, fmt.Errorf("parse vm.page_pageable_internal_count: %w", err)
	}

	purgeablePages := stats["Pages purgeable"]
	wiredPages := stats["Pages wired down"]
	compressedPages := stats["Pages occupied by compressor"]

	appBytes := (pageablePages - purgeablePages) * pageSize
	wiredBytes := wiredPages * pageSize
	compressedBytes := compressedPages * pageSize
	usedBytes := appBytes + wiredBytes + compressedBytes

	if usedBytes > total {
		usedBytes = total
	}
	availableBytes := total - usedBytes

	usage := memoryUsageFromTotalAvailable(total, availableBytes)
	usage.AppMemoryBytes = appBytes
	usage.WiredMemoryBytes = wiredBytes
	usage.CompressedMemoryBytes = compressedBytes
	usage.UsedBytes = usedBytes
	usage.AvailableBytes = availableBytes
	if total > 0 {
		usage.UsedPercent = float64(usedBytes) / float64(total) * 100
	}
	return usage, nil
}

func parseDarwinVmStat(output string) (map[string]uint64, uint64, error) {
	stats := map[string]uint64{}
	pageSize := uint64(4096)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, "page size of") {
			if idx := strings.Index(line, "page size of "); idx >= 0 {
				rest := line[idx+len("page size of "):]
				fields := strings.Fields(strings.TrimSuffix(rest, ")"))
				if len(fields) > 0 {
					n, err := strconv.ParseUint(strings.TrimSuffix(fields[0], "bytes"), 10, 64)
					if err == nil && n > 0 {
						pageSize = n
					}
				}
			}
			continue
		}

		colon := strings.Index(line, ":")
		if colon < 0 {
			continue
		}
		key := strings.Trim(strings.TrimSpace(line[:colon]), `"`)
		valStr := strings.TrimSpace(line[colon+1:])
		valStr = strings.TrimSuffix(valStr, ".")
		valStr = strings.TrimSpace(valStr)
		if valStr == "" {
			continue
		}
		n, err := strconv.ParseUint(valStr, 10, 64)
		if err != nil {
			continue
		}
		stats[key] = n
	}

	if len(stats) == 0 {
		return nil, 0, fmt.Errorf("no vm_stat counters parsed")
	}
	return stats, pageSize, nil
}

func darwinMemoryFreePercent() (float64, error) {
	out, err := exec.Command("/usr/bin/memory_pressure").Output()
	if err != nil {
		return 0, fmt.Errorf("memory_pressure: %w", err)
	}
	return parseDarwinMemoryFreePercent(string(out))
}

func parseDarwinMemoryFreePercent(output string) (float64, error) {
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, "System-wide memory free percentage") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			break
		}
		s := strings.TrimSpace(strings.TrimSuffix(parts[1], "%"))
		pct, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("parse memory free percentage: %w", err)
		}
		return pct, nil
	}
	return 0, fmt.Errorf("system-wide memory free percentage not found")
}

func memoryUsageWindows() (MemoryUsage, error) {
	ps := `$os = Get-CimInstance Win32_OperatingSystem; Write-Output "$($os.TotalVisibleMemorySize) $($os.FreePhysicalMemory)"`
	out, err := exec.Command("powershell", "-NoProfile", "-Command", ps).Output()
	if err != nil {
		return MemoryUsage{}, err
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 2 {
		return MemoryUsage{}, fmt.Errorf("unexpected Win32_OperatingSystem output: %q", string(out))
	}
	totalKB, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return MemoryUsage{}, fmt.Errorf("parse TotalVisibleMemorySize: %w", err)
	}
	freeKB, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return MemoryUsage{}, fmt.Errorf("parse FreePhysicalMemory: %w", err)
	}
	return memoryUsageFromTotalAvailable(totalKB*1024, freeKB*1024), nil
}
