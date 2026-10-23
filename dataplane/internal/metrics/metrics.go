package metrics

import (
	"net/http"
	_ "net/http/pprof"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/sithxxx/S1thVPN/dataplane/internal/device"
)

func Register(s *device.Server) {
	counter := func(name, help string, v *atomic.Uint64) {
		prometheus.MustRegister(prometheus.NewCounterFunc(
			prometheus.CounterOpts{Namespace: "s1thvpn", Name: name, Help: help},
			func() float64 { return float64(v.Load()) }))
	}
	counter("handshake_ok_total", "Успешные рукопожатия", &s.Stats.HandshakesOK)
	counter("handshakes_failed_total", "Отвергнутые рукопожатия", &s.Stats.HandshakesFailed)
	counter("handshakes_ratelimited_total", "Отброшено лимитом рукопожатий", &s.Stats.HandshakesRateLimited)
	counter("drop_decrypt_total", "Не прошли проверку AEAD", &s.Stats.DropDecrypt)
	counter("drop_replay_total", "Повторы", &s.Stats.DropReplay)
	counter("drop_spoof_total", "Подмена адреса источника", &s.Stats.DropSpoof)
	counter("drop_ratelimit_total", "Отброшено лимитом скорости", &s.Stats.DropRateL)
	counter("drop_nopeer_total", "Неизвестный получатель или индекс", &s.Stats.DropNoPeer)

	prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Namespace: "S1thVPN", Name: "peers_active", Help: "Пиры с рукопожатием за 180 с"},
		func() float64 {
			n := 0
			for _, p := range s.Peers.Peers() {
				if _, hs, _ := p.Info(); time.Since(hs) < 180*time.Second {
					n++
				}
			}
			return float64(n)
		}))
}

func Serve(addr string) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(addr, nil)
}
