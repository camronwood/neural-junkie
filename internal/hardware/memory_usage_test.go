package hardware

import (
	"strings"
	"testing"
)

func TestMemoryUsageFromTotalAvailable(t *testing.T) {
	u := memoryUsageFromTotalAvailable(32*1024*1024*1024, 8*1024*1024*1024)
	if u.TotalBytes != 32*1024*1024*1024 {
		t.Fatalf("total %d", u.TotalBytes)
	}
	if u.AvailableBytes != 8*1024*1024*1024 {
		t.Fatalf("available %d", u.AvailableBytes)
	}
	if u.UsedBytes != 24*1024*1024*1024 {
		t.Fatalf("used %d", u.UsedBytes)
	}
	if u.UsedPercent < 74.9 || u.UsedPercent > 75.1 {
		t.Fatalf("used_percent %v", u.UsedPercent)
	}
}

func TestParseLinuxMeminfo(t *testing.T) {
	sample := "MemTotal:       16384000 kB\nMemAvailable:    4096000 kB\n"
	total, available, err := parseLinuxMeminfo(strings.NewReader(sample))
	if err != nil {
		t.Fatal(err)
	}
	if total != 16384000*1024 {
		t.Fatalf("total %d", total)
	}
	if available != 4096000*1024 {
		t.Fatalf("available %d", available)
	}
}

func TestParseDarwinVmStat(t *testing.T) {
	sample := `Mach Virtual Memory Statistics: (page size of 16384 bytes)
Pages free:                               16035.
Pages active:                            282307.
Pages inactive:                          279819.
Pages wired down:                        689754.
Pages purgeable:                          19264.
"Pages occupied by compressor":            247867.
`
	stats, pageSize, err := parseDarwinVmStat(sample)
	if err != nil {
		t.Fatal(err)
	}
	if pageSize != 16384 {
		t.Fatalf("pageSize %d", pageSize)
	}
	if stats["Pages wired down"] != 689754 {
		t.Fatalf("wired %d", stats["Pages wired down"])
	}
	if stats["Pages occupied by compressor"] != 247867 {
		t.Fatalf("compressor %d", stats["Pages occupied by compressor"])
	}
}

func TestParseDarwinMemoryFreePercent(t *testing.T) {
	sample := "Some header\nSystem-wide memory free percentage: 42%\n"
	pct, err := parseDarwinMemoryFreePercent(sample)
	if err != nil {
		t.Fatal(err)
	}
	if pct != 42 {
		t.Fatalf("pct %v", pct)
	}
}

func TestMemoryUsageSnapshot(t *testing.T) {
	snap, err := MemoryUsageSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if snap.TotalBytes < 1 {
		t.Fatalf("total_bytes = %d", snap.TotalBytes)
	}
	if snap.UsedPercent < 0 || snap.UsedPercent > 100 {
		t.Fatalf("used_percent = %v", snap.UsedPercent)
	}
}
