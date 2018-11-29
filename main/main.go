package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/zjshen14/go-p2p"
)

var (
	ip            string
	port          int
	seed          int64
	secureIO      bool
	gossip        bool
	bootstrapAddr string
	frequency     int64
	slowStart     int64
)

var (
	receiveCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "broadcast_message_receive_audit",
			Help: "Broadcast message_receive_audit",
		},
		[]string{"from", "to"},
	)
	sendCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "broadcast_message_send_audit",
			Help: "Broadcast message_send_audit",
		},
		[]string{"from"},
	)
)

func init() {
	flag.StringVar(&ip, "ip", "127.0.0.1", "IP address")
	flag.IntVar(&port, "port", 30001, "Port number")
	flag.Int64Var(&seed, "seed", 0, "Seed to generate key")
	flag.BoolVar(&secureIO, "secureio", false, "Use secure I/O")
	flag.BoolVar(&gossip, "gossip", false, "Use Gossip protocol")
	flag.StringVar(&bootstrapAddr, "bootstrapaddr", "", "Bootstrap node address")
	flag.Int64Var(&frequency, "frequency", 1000, "How frequent (in ms) to send a message")
	flag.Int64Var(&slowStart, "slowstart", 10, "Wait some time (in sec) before sending a message")
	flag.Parse()

	prometheus.MustRegister(receiveCounter)
	prometheus.MustRegister(sendCounter)
}

func main() {
	if ipFromEnv, ok := os.LookupEnv("P2P_IP"); ok {
		ip = ipFromEnv
	}
	if portFromEnv, ok := os.LookupEnv("P2P_PORT"); ok {
		portIntFromEvn, err := strconv.Atoi(portFromEnv)
		if err != nil {
			p2p.Logger.Panic().Err(err).Msg("Error when parsing port number from ENV")
		}
		port = portIntFromEvn

	}
	options := []p2p.Option{
		p2p.IP(ip),
		p2p.Port(port),
		p2p.Seed(seed),
	}
	if secureIO {
		options = append(options, p2p.SecureIO())
	}
	if gossip {
		options = append(options, p2p.Gossip())
	}

	host, err := p2p.NewHost(context.Background(), options...)
	if err != nil {
		p2p.Logger.Panic().Err(err).Msg("Error when instantiating a host")
	}

	audit := make(map[string]int, 0)

	if err := host.AddPubSub("measurement", func(data []byte) error {
		id := string(data)
		if _, ok := audit[id]; ok {
			audit[id]++
		} else {
			audit[id] = 1
		}
		if audit[id]%10 == 0 {
			p2p.Logger.Info().Str("id", id).Int("num", audit[id]).Msg("Received messages")
		}
		receiveCounter.WithLabelValues(id, host.Identity()).Inc()
		return nil
	}); err != nil {
		p2p.Logger.Panic().Err(err).Msg("Error when adding pubsub")
	}

	if bootstrapAddr != "" {
		if err := host.Connect(bootstrapAddr); err != nil {
			p2p.Logger.Panic().Err(err).Msg("Error when connecting to the bootstrap node")
		}
		if err := host.JoinOverlay(); err != nil {
			p2p.Logger.Panic().Err(err).Msg("Error when joining the overlay")
		}
	}

	tick := time.Tick(time.Duration(frequency) * time.Millisecond)
	for {
		select {
		case <-tick:
			if err := host.Broadcast(
				"measurement",
				[]byte(fmt.Sprintf("%s", host.Identity())),
			); err != nil {
				p2p.Logger.Error().Err(err).Msg("Error when broadcasting a message")
			} else {
				sendCounter.WithLabelValues(host.Identity()).Inc()
			}
		}
	}
}
