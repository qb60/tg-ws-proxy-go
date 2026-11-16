//go:build !linux

package sockopts

import "testing"

func TestMarkControlUnsupported(t *testing.T) {
	if MarkControl(0) != nil {
		t.Fatal("zero mark should leave the dialer unchanged")
	}
	if err := MarkControl(1)("tcp4", "127.0.0.1:443", nil); err == nil {
		t.Fatal("nonzero mark must fail on unsupported platforms")
	}
}
