package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/zjshen14/go-p2p"
)

var (
	hostName      string
	port          int
	secureIO      bool
	gossip        bool
	bootstrapAddr string
	frequency     int64
	slowStart     int64
	broadcast     bool
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
	flag.StringVar(&hostName, "host", "127.0.0.1", "Host name or IP address")
	flag.IntVar(&port, "port", 30001, "Port number")
	flag.BoolVar(&secureIO, "secureio", false, "Use secure I/O")
	flag.BoolVar(&gossip, "gossip", false, "Use Gossip protocol")
	flag.StringVar(&bootstrapAddr, "bootstrapaddr", "", "Bootstrap node address")
	flag.Int64Var(&frequency, "frequency", 1000, "How frequent (in ms) to send a message")
	flag.Int64Var(&slowStart, "slowstart", 10, "Wait some time (in sec) before sending a message")
	flag.BoolVar(&broadcast, "broadcast", true, "Broadcast or unicast the messages only to the neighbors")
	flag.Parse()

	prometheus.MustRegister(receiveCounter)
	prometheus.MustRegister(sendCounter)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(":8080", mux); err != nil {
			p2p.Logger().Error().Err(err).Msg("Error when serving performance profiling data")
		}
	}()
}

func main() {
	if ipFromEnv, ok := os.LookupEnv("P2P_IP"); ok {
		hostName = ipFromEnv
	}
	if portFromEnv, ok := os.LookupEnv("P2P_PORT"); ok {
		portIntFromEvn, err := strconv.Atoi(portFromEnv)
		if err != nil {
			p2p.Logger().Panic().Err(err).Msg("Error when parsing port number from ENV")
		}
		port = portIntFromEvn

	}

	options := []p2p.Option{
		p2p.HostName(hostName),
		p2p.Port(port),
	}

	if secureIO {
		options = append(options, p2p.SecureIO())
	}
	if gossip {
		options = append(options, p2p.Gossip())
	}

	host, err := p2p.NewHost(context.Background(), options...)
	if err != nil {
		p2p.Logger().Panic().Err(err).Msg("Error when instantiating a host")
	}

	audit := make(map[string]int, 0)
	var mutex sync.Mutex

	handleMsg := func(data []byte) error {
		mutex.Lock()
		defer mutex.Unlock()

		id := string(data)
		if _, ok := audit[id]; ok {
			audit[id]++
		} else {
			audit[id] = 1
		}
		if audit[id]%100 == 0 {
			p2p.Logger().Info().Str("id", id).Int("num", audit[id]).Msg("Received messages")
		}
		receiveCounter.WithLabelValues(id, host.Address()).Inc()
		return nil
	}
	if err := host.AddBroadcastPubSub("measurement", handleMsg); err != nil {
		p2p.Logger().Panic().Err(err).Msg("Error when adding broadcast pubsub")
	}
	if err := host.AddUnicastPubSub("measurement", handleMsg); err != nil {
		p2p.Logger().Panic().Err(err).Msg("Error when adding unicast pubsub")
	}

	if bootstrapAddr != "" {
		if err := host.Connect(bootstrapAddr); err != nil {
			p2p.Logger().Panic().Err(err).Msg("Error when connecting to the bootstrap node")
		}
		if err := host.JoinOverlay(); err != nil {
			p2p.Logger().Panic().Err(err).Msg("Error when joining the overlay")
		}
	}

	tick := time.Tick(time.Duration(frequency) * time.Millisecond)
	for {
		select {
		case <-tick:
			var err error
			if broadcast {
				err = host.Broadcast("measurement", []byte(fmt.Sprintf("%s", host.Address())))
			} else {
				neighbors, err := host.Neighbors()
				if err != nil {
					p2p.Logger().Error().Err(err).Msg("Error when getting neighbors")
				}
				for _, neighbor := range neighbors {
					host.Unicast(neighbor, "measurement", []byte(fmt.Sprintf("%s", host.Address())))
				}
			}
			if err != nil {
				p2p.Logger().Error().Err(err).Msg("Error when broadcasting a message")
			} else {
				sendCounter.WithLabelValues(host.Address()).Inc()
			}
		}
	}
}
