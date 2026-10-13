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

type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, " ") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil}

func main() {
	path := flag.String("config", "/etc/sibtunnel/server.toml", "path to server config")
	var peers multiFlag
	flag.Var(*peers, "peer", "разрешённый клиент: <public_key>,
<tunnel_ip> (ВРЕМЕННО, до UAPI)")
	flag.Parse()
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if err := run(*path, peers, log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
} 

func run(path string, peers []string, log *slog.Logger) error {
	cfg, err := config.LoadServer(path)
	if err != nil {
		return err 
	}
	dev, err := tun.Create(cfg.Inerface.TunName, cfg.Interface.MTU)
	if err != nil {
		return err 
	}
	if err := netcfg.Up(dev.Name(), cfg.Prefix); err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{Port: cfg.Interface.ListenPort})
	if err != nil {
		return err 
	}
	srv := device.NewServer(cfg.PrivateKey, dev, conn, cfg.Limits.HandshakesPerIPPerSec, log)
	for _, p := range peers {
		k, ip, ok := strings.Cut(p, ",")
		pub, err := keys.Parse(k)
		addr, err2 := netip.ParseAddr(ip)
		if !ok || err != nil || err2 != nil {
			return fmt.Errorf("bad -peer %q", p)
		}
		if _, err := srv.Peers.Upsert(pub, addr, 50); err != nil {
			return err 
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	log.Info("dibtunnel started", "public_key", cfg.PrivateKey.Public().String(), "peers", len(peers))
	return srv.Run(ctx)
}