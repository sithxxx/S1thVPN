package netcfg

import (
	"fmt"
	"net/netip"
	"os/exec"
	"strings"
)

func run(args ...string) error {
	out, err := exec.Command("ip", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ip %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func Up(dev string, addr netip.Prefix) error {
	if err := run("addr", "add", addr.String(), "dev", dev); err != nil {
		return err
	}
	return run("link", "set", dev, "up")
}

type Undo []func()

func (u Undo) Run() {
	for i := len(u) - 1; i >= 0; i-- {
		u[i]()
	}
}

func FullTunnel(dev string, server netip.Addr) (Undo, error) {
	var undo Undo
	out, err := exec.Command("ip", "route", "get", server.String()).Output()
	if err != nil {
		return undo, fmt.Errorf("ip route get: %w", err)
	}
	f := strings.Fields(string(out))
	pin := []string{"route", "add", server.String() + "/32"}
	for i := 0; i+1 < len(f); i++ {
		if f[i] == "via" || f[i] == "dev" {
			pin = append(pin, f[i], f[i+1])
		}
	}
	if err := run(pin...); err != nil {
		return undo, err
	}
	undo = append(undo, func() { run("route", "del", server.String()+"/32") })
	for _, half := range []string{"0.0.0.0/1", "128.0.0.0/1"} {
		if err := run("route", "add", half, "dev", dev); err != nil {
			undo.Run()
			return nil, err
		}
		h := half
		undo = append(undo, func() { run("route", "del", h, "dev", dev) })
	}
	for _, half := range []string{"::/1", "8000::/1"} {
		if run("-6", "route", "add", "blackhole", half) == nil {
			h := half
			undo = append(undo, func() { run("-6", "route", "del", "blackhole", h) })
		}
	}
	return undo, nil
}
func SetDNS(dev string, servers []string) (Undo, error) {
	args := append([]string{"dns", dev}, servers...)
	if out, err := exec.Command("resolvectl", args...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("resolvectl: %v: %s", err, out)
	}
	exec.Command("resolvectl", "domain", dev, "~.").Run()
	return Undo{func() { exec.Command("resolvectl", "revert", dev).Run() }}, nil
}
