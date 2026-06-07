package hardware

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// TotalMemoryBytes returns installed physical RAM. Returns 0 when detection fails.
func TotalMemoryBytes() (uint64, error) {
	switch runtime.GOOS {
	case "linux":
		return totalMemoryLinux()
	case "darwin":
		return totalMemoryDarwin()
	case "windows":
		return totalMemoryWindows()
	default:
		return 0, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func totalMemoryLinux() (uint64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			break
		}
		kb, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, err
		}
		return kb * 1024, nil
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, fmt.Errorf("MemTotal not found in /proc/meminfo")
}

func totalMemoryDarwin() (uint64, error) {
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0, err
	}
	s := strings.TrimSpace(string(out))
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse hw.memsize: %w", err)
	}
	return n, nil
}

func totalMemoryWindows() (uint64, error) {
	// PowerShell one-liner avoids cgo / x/sys dependency for GlobalMemoryStatusEx.
	ps := `[Math]::Round((Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory)`
	out, err := exec.Command("powershell", "-NoProfile", "-Command", ps).Output()
	if err != nil {
		return 0, err
	}
	s := strings.TrimSpace(string(out))
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse TotalPhysicalMemory: %w", err)
	}
	return n, nil
}

// MemoryGB rounds total bytes down to whole gigabytes for tier selection.
func MemoryGB(totalBytes uint64) int {
	if totalBytes == 0 {
		return 0
	}
	return int(totalBytes / (1024 * 1024 * 1024))
}
