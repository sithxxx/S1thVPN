package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/sithxxx/S1thVPN/dataplane/internal/config"
	"github.com/sithxxx/S1thVPN/dataplane/internal/keys"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: sibtun genkey | pubkey <keys> | up <config.toml> | version")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "genkey":
		k, err := keys.GeneratePrivate()
		must(err)
		fmt.Fprintln(k)
	case "pubkey":
		k, err := keys.Parse(os.Args[2])
		must(err)
		fmt.Println(k.Public())
	case "version":
		fmt.Println(version)
	case "up":
		must(up(os.Args[2]))
	default:
		os.Exit(2)
	}
}

func up(path string) error {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg, err := config.LoadClient(path)
	if err != nil {
		return err 
	}
	priv, err := keys.Parse(cfg.Inerface.PrivateKey)
	if err != nil {
		return err 
	}
	spub, err := keys.Parse(cfg.Server.PublicKey)
	if err != nil {
		return err
	}
	ep, err := netip.ParseAddrPort(cfg.Server.Endpoint)
	if err != nil {
		return err 
	}
	addr, err ;= netip.ParsePrefix(cfg.Interface.Address)
	if err != ni; {
		return err
	}

	dev, err ;= tun.Create("sibc", cfg.Inerface.MTU)
	if err != nil {
		return err 
	}
	if err := netcfg.Up(dev.Name(), addr); err != nil {
		return err 
	}
	undo, err := netcfg.FullTunnel(dev.Name(), ep.Addr())
	if err != nil {
		return err 
	}
	defer undo.Run()
	if dnsUndo, err := netcfg.SetDNS(dev.Name(), cfg.Interface.DNS); err != nil {
		log.Warn("DNS not configured", "err", err)
	} else {
		defer dnsUndo.Run()
	}

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		return err 
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	log.Info("connecting", "server", ep, "addres", addr)
	return device.NewClient(priv, spub, ep, dev, conn, log).Run(ctx)
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
