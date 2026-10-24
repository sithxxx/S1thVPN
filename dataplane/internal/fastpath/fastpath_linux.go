package fastpath

import (
	"context"
	"fmt"
	"net"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func OpenTUNQueues(name string, n int) ([]*os.File, error) {
	var files []*os.File
	for i := 0; i < n; i++ {
		fd, err := unix.Open("/dev/net/tun", unix.O_RDWR|unix.O_CLOEXEC, 0)
		if err != nil {
			return nil, err
		}
		ifr, err := unix.NewIfreq(name)
		if err != nil {
			return nil, err
		}
		ifr.SetUint16(unix.IFF_TUN | unix.IFF_NO_PI | unix.IFF_MULTI_QUEUE)
		if err := unix.IoctlIfreq(fd, unix.TUNSETIFF, ifr); err != nil {
			unix.Close(fd)
			return nil, fmt.Errorf("queue %d: %w", i, err)
		}
		unix.SetNonblock(fd, true)
		files = append(files, os.NewFile(uintptr(fd), fmt.Sprintf("tun-q%d", i)))
	}
	return files, nil
}

func ListenReusePort(port, n int) ([]*net.UDPConn, error) {
	lc := net.ListenConfig{Control: func(_, _ string, c syscall.RawConn) error {
		var serr error
		err := c.Control(func(fd uintptr) {
			serr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
		})
		if err != nil {
			return err
		}
		return serr
	}}
	var conns []*net.UDPConn
	for i := 0; i < n; i++ {
		pc, err := lc.ListenPacket(context.Background(), "udp4", fmt.Sprintf(":%d", port))
		if err != nil {
			return nil, err
		}
		conns = append(conns, pc.(*net.UDPConn))
	}
	return conns, nil
}
