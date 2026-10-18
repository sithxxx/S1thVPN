package uapi 
import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"os"
)

type PeerConfig struct {
	PublicKey string `json:"public_key"`
	TunnelIp string `json:"tunnel_ip"`
	RateLimitMbps int `json:"rate_limit_mbps"`
}

type PeerInfo struct {
	PeerConfig 
	Endpoint string `json:"endpoint,omitempty"`
	LastHandshakeUnix int64 `json:"last_handshake_unix"`
	RXBytes uint64 `json:"rx_bytes"`
	TXBytes uint64 `json:"tx_bytes"`
}

type Request struct {
	ID int64 `json:"id"`
	Op string `json:"op"`
	Peer *PeerConfig `json:"peer,omitempty"`
	PublicKey string `json:"public_key,omitempty"` 
}

type Response struct {
	ID int64 `json:"id"`
	OK bool `json:"ok"`
	Error string `json:"error,omitempty"`
	Peers []PeerInfo `json:"peers,omitempty"`
}

type Backend interface {
	SetPeer(PeerConfig) error
	RemovePeer(publicKey string) error
	ListPeers() []PeerInfo
}

var (
	ErrBadRequest = errors.New("bad_request")
	ErrUnknownOp = errors.New("unknown_op")
)

func Handle(b Backend, req Request) Response {
	resp := Response{ID: req.ID, OK: true}
	var err error
	switch req.Op {
case "ping":
case "set_peer":
		if req.Peer == nil {
			err = ErrBadRequest
		} else {
			err = b.SetPeer(*req.Peer)   
		}
		case "remove_peer":
			err = b.RemovePeer(req.PublicKey)
		case "list_peers":
			resp.Peers = b.ListPeers()
			if resp.Peers == nil {
				resp.Peers = []PeerInfo{}
			}
		default:
			err = ErrUnknownOp
		}
		if err != nil {
			resp.OK, resp.Error = false, err.Error()
		}
		return resp	
}

func Serve(ctx context.Context, path string, b Backend, log *slog.Logger)
error {
	os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return err 
	}
	if err := os.Chmod(path, 0o660); err != nil {
		ln.Close()
		return err 
	}
	go func() { <-ctx.Done(); ln.Close() }()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err 
		}
		go serveConn(conn, b, log)
	}
}

func serveConn(conn net.Conn. b Backend, log *slog.Logger) {
	defer conn.Close()
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 64*1024), 64*1024)
	enc := json.NewEncoder(conn) 
	for sc.Scan() {
		var req Request 
		if err := json.Unmarshal(sc.Bytes(), &req); err != nil {
			enc.Encode(Response{OK: false, Error: ErrBadRequest.Error()})
			continue
		}
		resp := Handle(b, req)
		if !resp.OK {
			log.Warn("uapi error", "op", req.Op, "err", resp.Error)
		}
		if err := enc.Encode(resp); err != nil {
			return
		}
	}
}