package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

const (
	// ProtocolID is the protocol identifier for the blockchain
	ProtocolID = "/layer1/1.0.0"
	// MaxMessageSize is the maximum size of a P2P message
	MaxMessageSize = 10 * 1024 * 1024 // 10 MB
)

// Network manages P2P networking
type Network struct {
	host      host.Host
	dht       *dht.IpfsDHT
	ctx       context.Context
	cancel    context.CancelFunc
	
	handlers  map[MessageType]MessageHandler
	mu        sync.RWMutex
	
	peers     map[peer.ID]bool
	peersMu   sync.RWMutex
}

// MessageHandler is a function that handles incoming messages
type MessageHandler func(peerID peer.ID, msg *Message) error

// Config contains network configuration
type Config struct {
	ListenAddr    string
	BootstrapPeers []string
}

// NewNetwork creates a new P2P network
func NewNetwork(ctx context.Context, cfg *Config) (*Network, error) {
	ctx, cancel := context.WithCancel(ctx)
	
	// Create libp2p host
	listenAddr, err := multiaddr.NewMultiaddr(cfg.ListenAddr)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("invalid listen address: %w", err)
	}
	
	h, err := libp2p.New(
		libp2p.ListenAddrs(listenAddr),
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create host: %w", err)
	}
	
	// Create DHT for peer discovery
	kdht, err := dht.New(ctx, h)
	if err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to create DHT: %w", err)
	}
	
	n := &Network{
		host:     h,
		dht:      kdht,
		ctx:      ctx,
		cancel:   cancel,
		handlers: make(map[MessageType]MessageHandler),
		peers:    make(map[peer.ID]bool),
	}
	
	// Set stream handler
	h.SetStreamHandler(protocol.ID(ProtocolID), n.handleStream)
	
	// Bootstrap DHT
	if err := kdht.Bootstrap(ctx); err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to bootstrap DHT: %w", err)
	}
	
	// Connect to bootstrap peers
	if len(cfg.BootstrapPeers) > 0 {
		go n.connectToBootstrapPeers(cfg.BootstrapPeers)
	}
	
	return n, nil
}

// RegisterHandler registers a message handler
func (n *Network) RegisterHandler(msgType MessageType, handler MessageHandler) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.handlers[msgType] = handler
}

// Broadcast broadcasts a message to all connected peers
func (n *Network) Broadcast(msg *Message) error {
	n.peersMu.RLock()
	peers := make([]peer.ID, 0, len(n.peers))
	for p := range n.peers {
		peers = append(peers, p)
	}
	n.peersMu.RUnlock()
	
	data, err := msg.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}
	
	for _, p := range peers {
		go func(peerID peer.ID) {
			if err := n.sendToPeer(peerID, data); err != nil {
				// Log error but don't fail the broadcast
				fmt.Printf("Failed to send to peer %s: %v\n", peerID, err)
			}
		}(p)
	}
	
	return nil
}

// SendToPeer sends a message to a specific peer
func (n *Network) SendToPeer(peerID peer.ID, msg *Message) error {
	data, err := msg.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}
	return n.sendToPeer(peerID, data)
}

// sendToPeer sends raw data to a peer
func (n *Network) sendToPeer(peerID peer.ID, data []byte) error {
	stream, err := n.host.NewStream(n.ctx, peerID, protocol.ID(ProtocolID))
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}
	defer stream.Close()
	
	// Set write deadline
	stream.SetWriteDeadline(time.Now().Add(30 * time.Second))
	
	// Write message
	if _, err := stream.Write(data); err != nil {
		return fmt.Errorf("failed to write to stream: %w", err)
	}
	
	return nil
}

// handleStream handles incoming streams
func (n *Network) handleStream(stream network.Stream) {
	defer stream.Close()
	
	peerID := stream.Conn().RemotePeer()
	
	// Add peer to known peers
	n.peersMu.Lock()
	n.peers[peerID] = true
	n.peersMu.Unlock()
	
	// Read message
	buf := make([]byte, MaxMessageSize)
	bytesRead, err := stream.Read(buf)
	if err != nil {
		fmt.Printf("Failed to read from stream: %v\n", err)
		return
	}
	
	// Deserialize message
	msg, err := DeserializeMessage(buf[:bytesRead])
	if err != nil {
		fmt.Printf("Failed to deserialize message: %v\n", err)
		return
	}
	
	// Handle message
	n.mu.RLock()
	handler, exists := n.handlers[msg.Type]
	n.mu.RUnlock()
	
	if exists {
		if err := handler(peerID, msg); err != nil {
			fmt.Printf("Handler error: %v\n", err)
		}
	}
}

// connectToBootstrapPeers connects to bootstrap peers
func (n *Network) connectToBootstrapPeers(bootstrapPeers []string) {
	for _, addrStr := range bootstrapPeers {
		addr, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			fmt.Printf("Invalid bootstrap address %s: %v\n", addrStr, err)
			continue
		}
		
		peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			fmt.Printf("Failed to parse peer info: %v\n", err)
			continue
		}
		
		if err := n.host.Connect(n.ctx, *peerInfo); err != nil {
			fmt.Printf("Failed to connect to bootstrap peer %s: %v\n", peerInfo.ID, err)
		} else {
			fmt.Printf("Connected to bootstrap peer %s\n", peerInfo.ID)
		}
	}
}

// PeerCount returns the number of connected peers
func (n *Network) PeerCount() int {
	n.peersMu.RLock()
	defer n.peersMu.RUnlock()
	return len(n.peers)
}

// ID returns the host's peer ID
func (n *Network) ID() peer.ID {
	return n.host.ID()
}

// Addrs returns the host's listen addresses
func (n *Network) Addrs() []multiaddr.Multiaddr {
	return n.host.Addrs()
}

// Close closes the network
func (n *Network) Close() error {
	n.cancel()
	if err := n.dht.Close(); err != nil {
		return err
	}
	return n.host.Close()
}

