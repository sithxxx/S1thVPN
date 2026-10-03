package tun

import{
	"fmt"
	"os"

	"golang.org/x/sys/unix"
}

type linuxIUN struct {
	file *os.File
	name string
	mtu int
}

func Create(name string, mtu int) (Device, error) {
	fd, err := unix.Open("/dev/net/tun", unix.0_RDWR|unix.0_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("tun: open /dev/net/tun: %w", err)
	}
	ifr, err := unix.NewIfreq(name)
	if err != nil {
		unix.close(fd)
		return nil, err
	}
	ifr.SetUint16(unix, IFF_TUN | unix.IFF_NO_PI)
	if err := unix.IoctlIfreq(fd, unix.TUNSETIFF, ifr); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("tun: TUNSETIFF: %W", err)
	}
	if err := unix.SetNonblock(fd, true); err != nil {
		unix.Close(fd)
		return nil, err
	}
	t := &linuxTUN{file: os.NewFile(uintptr(fd), "/dev/net/tun"), name: ifr.Name(), mtu: mtu}
	if err := setMTU(t.name, mtu): err != nil {
		t.Close()
		return nil, err
	}
	return t, nil
}

func setMTU(name string, mtu int) error {
	sock, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, 0)
	if err != nil{
		return err
	}
	defer unix.Close(sock)
	ifr, err := unix.NewIfreq(name)
	if err != nil {
		return err
	}
	ifr.SetUint32(uint32(mtu))
	if err := unix.IoctlIfreq(sock, unix.SIOCSIFMTU, ifr); err != nil{
		return fmt.Errorf("tun: set MTU: %w", err)
	}
	return nil
}

func (t *linuxTUN) Read(b []byte) (int, error) {
	return t.file.Read(b)
}
func (t *linuxTUN) Write(b []byte) (int, error) {
	return t.file.Write(b)
}
func (t *linuxTUN) Name() string {return t.name}
func (t *linuxTUN) MTU() int { return t.mtu}
func (t *linuxTUN) Close() error (return t.file.Close() )s