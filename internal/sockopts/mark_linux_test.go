package sockopts

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"
	"time"
)

func TestMarkControlDisabled(t *testing.T) {
	if MarkControl(0) != nil {
		t.Fatal("zero mark should leave the dialer unchanged")
	}
}

func TestMarkControlSocketAndAcceptedConnection(t *testing.T) {
	for _, mark := range []uint32{0x100, 0xffffffff} {
		listener, err := net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { listener.Close() })
		dialer := net.Dialer{Control: MarkControl(mark)}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		t.Cleanup(cancel)
		conn, err := dialer.DialContext(ctx, "tcp4", listener.Addr().String())
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
			t.Skipf("SO_MARK requires capabilities: %v", err)
		}
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Close() })
		peer, err := listener.Accept()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { peer.Close() })
		for _, tt := range []struct {
			conn net.Conn
			want uint32
		}{{conn, mark}, {peer, 0}} {
			raw, err := tt.conn.(syscall.Conn).SyscallConn()
			if err != nil {
				t.Fatal(err)
			}
			var got int
			var sockErr error
			if err := raw.Control(func(fd uintptr) {
				got, sockErr = syscall.GetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_MARK)
			}); err != nil {
				t.Fatal(err)
			}
			if sockErr != nil || uint32(got) != tt.want {
				t.Fatalf("SO_MARK = %#x, %v; want %#x", uint32(got), sockErr, tt.want)
			}
		}
	}
}

type failedRawConn struct{ controlErr error }

func (c failedRawConn) Control(f func(uintptr)) error {
	if c.controlErr != nil {
		return c.controlErr
	}
	f(^uintptr(0)) // Invalid descriptor makes setsockopt fail.
	return nil
}
func (failedRawConn) Read(func(uintptr) bool) error  { panic("unused") }
func (failedRawConn) Write(func(uintptr) bool) error { panic("unused") }

func TestMarkControlPropagatesErrors(t *testing.T) {
	control := MarkControl(0x100)
	for _, want := range []error{syscall.EPERM, syscall.EBADF} {
		var raw failedRawConn
		if want == syscall.EPERM {
			raw.controlErr = want
		}
		if err := control("tcp4", "127.0.0.1:443", raw); !errors.Is(err, want) {
			t.Fatalf("got %v, want %v", err, want)
		}
	}
}
