//go:build !linux

package sockopts

import (
	"fmt"
	"syscall"
)

func MarkControl(mark uint32) func(string, string, syscall.RawConn) error {
	if mark == 0 {
		return nil
	}
	return func(_, _ string, _ syscall.RawConn) error {
		return fmt.Errorf("SO_MARK is only supported on Linux")
	}
}
