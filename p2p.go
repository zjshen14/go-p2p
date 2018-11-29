package p2p

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
)

// HandleBroadcast defines the callback function triggered when a broadcast message reaches a host
type HandleBroadcast func(data []byte) error

// Config enumerates the configs required by a host
type Config struct {
	IP       string
	Port     int
	Seed     int64
	SecureIO bool
	Gossip   bool
}

// DefaultConfig is a set of default configs
var DefaultConfig = Config{
	IP:       "127.0.0.1",
	Port:     30001,
	Seed:     0,
	SecureIO: false,
	Gossip:   false,
}

// Option defines the option function to modify the config for a host
type Option func(cfg *Config) error

// IP is the option to override the IP address
func IP(ip string) Option {
	return func(cfg *Config) error {
		cfg.IP = ip
		return nil
	}
}

// Port is the option to override the port number
func Port(port int) Option {
	return func(cfg *Config) error {
		cfg.Port = port
		return nil
	}
}

// Seed is the option to set the seed for generating key
func Seed(seed int64) Option {
	return func(cfg *Config) error {
		cfg.Seed = seed
		return nil
	}
}

// SecureIO is to indicate using secured I/O
func SecureIO() Option {
	return func(cfg *Config) error {
		cfg.SecureIO = true
		return nil
	}
}

// Gossip is to indicate using gossip protocol
func Gossip() Option {
	return func(cfg *Config) error {
		cfg.Gossip = true
		return nil
	}
}

// Host is the main struct that represents a host that communicating with the rest of the P2P networks
type Host struct {
	host      host.Host
	ctx       context.Context
	kad       *dht.IpfsDHT
	newPubSub func(ctx context.Context, h host.Host, opts ...pubsub.Option) (*pubsub.PubSub, error)
	pubs      map[string]*pubsub.PubSub
	subs      map[string]*pubsub.Subscription
	close     chan interface{}
}

// NewHost constructs a host struct
func NewHost(ctx context.Context, options ...Option) (*Host, error) {
	cfg := DefaultConfig
	for _, option := range options {
		if err := option(&cfg); err != nil {
			return nil, err
		}
	}

	r := rand.New(rand.NewSource(cfg.Seed))
	sk, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/%s/tcp/%d", cfg.IP, cfg.Port)),
		libp2p.Identity(sk),
	}

	if !cfg.SecureIO {
		opts = append(opts, libp2p.NoSecurity)
	}

	host, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}
	kad, err := dht.New(ctx, host)
	if err != nil {
		return nil, err
	}
	newPubSub := pubsub.NewFloodSub
	if cfg.Gossip {
		newPubSub = pubsub.NewGossipSub
	}
	myHost := Host{
		host:      host,
		ctx:       ctx,
		kad:       kad,
		newPubSub: newPubSub,
		pubs:      make(map[string]*pubsub.PubSub),
		subs:      make(map[string]*pubsub.Subscription),
		close:     make(chan interface{}),
	}
	Logger.Info().Str("address", myHost.Address()).Msg("P2P host started")
	return &myHost, nil
}

// Connect connects a peer given the address string
func (h *Host) Connect(address string) error {
	ma, err := multiaddr.NewMultiaddr(address)
	if err != nil {
		return err
	}
	target, err := peerstore.InfoFromP2pAddr(ma)
	if err != nil {
		return err
	}
	return h.host.Connect(h.ctx, *target)
}

// JoinOverlay triggers the host to join the DHT overlay
func (h *Host) JoinOverlay() error {
	v1b := cid.V1Builder{Codec: cid.Raw, MhType: multihash.SHA2_256}
	cid, err := v1b.Sum([]byte(h.Identity()))
	if err != nil {
		return err
	}
	if err := h.kad.Provide(h.ctx, cid, true); err != nil {
		return h.kad.Bootstrap(h.ctx)
	}
	return nil
}

// AddPubSub adds a topic that the host will pay attention to. This need to be called before using Connect/JoinOverlay.
// Otherwise, pubsub may not be aware of the existing overlay topology
func (h *Host) AddPubSub(topic string, callback HandleBroadcast) error {
	if _, ok := h.pubs[topic]; ok {
		return nil
	}
	pub, err := h.newPubSub(h.ctx, h.host)
	if err != nil {
		return err
	}
	sub, err := pub.Subscribe(topic)
	if err != nil {
		return err
	}
	h.pubs[topic] = pub
	h.subs[topic] = sub
	go func() {
		for {
			select {
			case <-h.close:
				return
			default:
				msg, err := sub.Next(h.ctx)
				if err != nil {
					Logger.Error().Err(err).Msg("Error when subscribing a broadcast message")
					continue
				}
				if err := callback(msg.Data); err != nil {
					Logger.Error().Err(err).Msg("Error when processing a broadcast message")
				}
			}
		}
	}()
	return nil
}

// Broadcast sends a message to the hosts who subscribe the topic
func (h *Host) Broadcast(topic string, data []byte) error {
	pub, ok := h.pubs[topic]
	if !ok {
		return nil
	}
	return pub.Publish(topic, data)
}

// Identity returns the identity string
func (h *Host) Identity() string {
	return h.host.ID().Pretty()
}

// IdentityHash returns the identity hash string
func (h *Host) IdentityHash() (string, error) {
	v1b := cid.V1Builder{Codec: cid.Raw, MhType: multihash.SHA2_256}
	cid, err := v1b.Sum([]byte(h.Identity()))
	if err != nil {
		return "", err
	}
	return cid.String(), nil
}

// Address returns the address
func (h *Host) Address() string {
	addr := h.host.Addrs()[0]
	if addr.String() == "/p2p-circuit" {
		addr = h.host.Addrs()[1]
	}
	hostAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ipfs/%s", h.Identity()))
	return addr.Encapsulate(hostAddr).String()
}

// Close closes the host
func (h *Host) Close() error {
	close(h.close)
	for _, sub := range h.subs {
		sub.Cancel()
	}
	if err := h.kad.Close(); err != nil {
		return err
	}
	if err := h.host.Close(); err != nil {
		return err
	}
	return nil
}
