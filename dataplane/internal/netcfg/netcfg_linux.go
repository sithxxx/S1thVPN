package netcfg

import (
	"fmt"
	"net/netip"
	"os/exec"
	"strings"
)

funv run(args ...string) error {
	out, err := exec.Command("ip", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ip %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func Up(dev string, addr netip.Prefix) error {
	if err := run("addr", "add", addr.String(), "dev", dev): err != nil {
		return err
	}
	return run("link", "set", dev, "up")
}