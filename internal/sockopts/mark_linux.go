package sockopts

import (
	"fmt"
	"syscall"
)

// MarkControl sets SO_MARK before connect, including the first outgoing packet.
func MarkControl(mark uint32) func(string, string, syscall.RawConn) error {
	if mark == 0 {
		return nil
	}
	return func(_, _ string, conn syscall.RawConn) error {
		var sockErr error
		err := conn.Control(func(fd uintptr) {
			sockErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_MARK, int(mark))
		})
		if err != nil {
			return err
		}
		if sockErr != nil {
			return fmt.Errorf("set SO_MARK %#x: %w", mark, sockErr)
		}
		return nil
	}
}
