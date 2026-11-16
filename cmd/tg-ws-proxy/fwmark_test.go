package main

import (
	"runtime"
	"strings"
	"testing"
)

func TestParseArgsFWMark(t *testing.T) {
	pa, err := parseArgs([]string{"--fwmark", "0x100"})
	if runtime.GOOS != "linux" {
		if err == nil {
			t.Fatal("expected unsupported platform error")
		}
		return
	}
	if err != nil || pa.cfg.FWMark != 256 {
		t.Fatalf("mark = %d, err = %v", pa.cfg.FWMark, err)
	}
	if !strings.Contains(startupSummary(pa), "fwmark=0x100") {
		t.Fatal("startup summary is missing the mark")
	}
	if _, err := parseArgs([]string{"--fwmark", "4294967296"}); err == nil {
		t.Fatal("expected overflow error")
	}
}

func TestProbeUpstreamRejectsInvalidFWMark(t *testing.T) {
	if got := probeUpstream([]string{"--fwmark", "-1", "127.0.0.1:443"}); got != 2 {
		t.Fatalf("exit code = %d; want 2", got)
	}
}
