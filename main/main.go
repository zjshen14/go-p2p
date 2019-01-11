package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/zjshen14/go-p2p"
)

var (
	hostName      string
	port          int
	extHostName   string
	httpPort      string
	extPort       int
	secureIO      bool
	gossip        bool
	bootstrapAddr string
	slowStart     int64
	broadcast     bool
)

func init() {
	flag.StringVar(&hostName, "host", "127.0.0.1", "Host name or IP address")
	flag.IntVar(&port, "port", 32221, "Port number")
	flag.StringVar(&extHostName, "exthost", "", "Host name or IP address seen from external")
	flag.IntVar(&extPort, "extport", 32221, "Port number seen from external")
	flag.BoolVar(&secureIO, "secureio", true, "Use secure I/O")
	flag.BoolVar(&gossip, "gossip", true, "Use Gossip protocol")
	flag.StringVar(&bootstrapAddr, "bootstrapaddr", "", "Bootstrap node address")
	flag.StringVar(&httpPort, "httpPort", ":8080", "httpPort")
	flag.Int64Var(&slowStart, "slowstart", 10, "Wait some time (in sec) before sending a message")
	flag.BoolVar(&broadcast, "broadcast", true, "Broadcast or unicast the messages only to the neighbors")
	flag.Parse()
}

func main() {
	if ipFromEnv, ok := os.LookupEnv("P2P_IP"); ok {
		hostName = ipFromEnv
	}
	if portFromEnv, ok := os.LookupEnv("P2P_PORT"); ok {
		portIntFromEvn, err := strconv.Atoi(portFromEnv)
		if err != nil {
			log.Panicln("Error when parsing port number from ENV, ", err)
		}
		port = portIntFromEvn
	}

	options := []p2p.Option{
		p2p.HostName(hostName),
		p2p.Port(port),
		p2p.ExternalHostName(extHostName),
		p2p.ExternalPort(extPort),
	}

	if secureIO {
		options = append(options, p2p.SecureIO())
	}
	if gossip {
		options = append(options, p2p.Gossip())
	}

	host, err := p2p.NewHost(context.Background(), options...)
	if err != nil {
		log.Panicln("Error when instantiating a host: ", err)
	}

	handleMsg := func(data []byte) error {
		log.Println("Get message: ", string(data))
		return nil
	}

	if err := host.AddBroadcastPubSub("measurement", handleMsg); err != nil {
		log.Panicln("Error when adding broadcast pubsub: ", err)
	}
	if err := host.AddUnicastPubSub("measurement", handleMsg); err != nil {
		log.Panicln("Error when adding unicast pubsub: ", err)
	}

	if bootstrapAddr != "" {
		if err := host.Connect(bootstrapAddr); err != nil {
			log.Panicln("Error when connecting to the bootstrap node: ", err)
		}
		if err := host.JoinOverlay(); err != nil {
			log.Panicln("Error when joining the overlay: ", err)
		}
	}

	http.HandleFunc("/unicast", func(w http.ResponseWriter, r *http.Request) {
		req := struct {
			Address string `json:"address"`
			Message string `json:"message"`
		}{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Fatalln("json unmarhsal failure: ", err)
		}
		if err := host.Unicast(req.Address, "measurement", []byte(req.Message)); err != nil {
			log.Panicln("unicast failure: ", err)
		}
	})

	http.HandleFunc("/broadcast", func(w http.ResponseWriter, r *http.Request) {
		req := struct {
			Message string `json:"message"`
		}{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Fatalln("json unmarhsal failure: ", err)
		}
		if err := host.Broadcast("measurement", []byte(req.Message)); err != nil {
			log.Panicln("broadcast failure: ", err)
		}
	})

	if err := http.ListenAndServe(httpPort, nil); err != nil {
		log.Fatalln("http listen failure: ", err)
	}
}
