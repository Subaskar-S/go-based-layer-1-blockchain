package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blockchain/layer1/internal/blockchain"
	"github.com/blockchain/layer1/internal/consensus"
	"github.com/blockchain/layer1/internal/crypto"
	"github.com/blockchain/layer1/internal/p2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/cobra"
)

var (
	dataDir      string
	httpPort     int
	grpcPort     int
	p2pPort      int
	metricsPort  int
	isValidator  bool
	validatorKey string
	bootstrapNodes []string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "layer1-node",
		Short: "Layer-1 Blockchain Node",
		Long:  "A production-ready Layer-1 blockchain node with PoS consensus and WASM smart contracts",
	}

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the blockchain node",
		RunE:  runNode,
	}

	startCmd.Flags().StringVar(&dataDir, "data-dir", "./data", "Data directory")
	startCmd.Flags().IntVar(&httpPort, "http-port", 8545, "HTTP RPC port")
	startCmd.Flags().IntVar(&grpcPort, "grpc-port", 9090, "gRPC port")
	startCmd.Flags().IntVar(&p2pPort, "p2p-port", 30303, "P2P port")
	startCmd.Flags().IntVar(&metricsPort, "metrics-port", 6060, "Metrics port")
	startCmd.Flags().BoolVar(&isValidator, "validator", false, "Run as validator")
	startCmd.Flags().StringVar(&validatorKey, "validator-key", "", "Validator private key file")
	startCmd.Flags().StringSliceVar(&bootstrapNodes, "bootstrap-nodes", []string{}, "Bootstrap nodes")

	rootCmd.AddCommand(startCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runNode(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("Starting Layer-1 Blockchain Node...")
	fmt.Printf("Data directory: %s\n", dataDir)
	fmt.Printf("HTTP RPC port: %d\n", httpPort)
	fmt.Printf("gRPC port: %d\n", grpcPort)
	fmt.Printf("P2P port: %d\n", p2pPort)
	fmt.Printf("Validator mode: %v\n", isValidator)

	// Create data directory
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Load or generate validator key
	var privKey *crypto.PrivateKey
	if isValidator {
		var err error
		privKey, err = loadOrGenerateKey(validatorKey)
		if err != nil {
			return fmt.Errorf("failed to load validator key: %w", err)
		}
		pubKey := privKey.Public()
		fmt.Printf("Validator address: %s\n", pubKey.Address().String())
	}

	// Create initial validator set (for demo, use hardcoded validators)
	validatorSet := createInitialValidatorSet(privKey)

	// Initialize blockchain
	bc, err := blockchain.NewBlockchain(dataDir, validatorSet)
	if err != nil {
		return fmt.Errorf("failed to initialize blockchain: %w", err)
	}
	defer bc.Close()

	fmt.Printf("Genesis hash: %s\n", bc.GetBestBlock().Hash.String())
	fmt.Printf("Current height: %d\n", bc.GetCurrentHeight())

	// Initialize P2P network
	p2pConfig := &p2p.Config{
		ListenAddr:     fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", p2pPort),
		BootstrapPeers: bootstrapNodes,
	}

	network, err := p2p.NewNetwork(ctx, p2pConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize P2P network: %w", err)
	}
	defer network.Close()

	fmt.Printf("P2P ID: %s\n", network.ID().String())
	fmt.Printf("P2P addresses: %v\n", network.Addrs())

	// Initialize consensus engine
	consensusEngine := consensus.NewEngine(ctx, validatorSet, privKey)
	consensusEngine.Start(bc.GetCurrentHeight())
	defer consensusEngine.Stop()

	// Register P2P message handlers
	registerP2PHandlers(network, bc, consensusEngine)

	// Start block production loop (if validator)
	if isValidator {
		go blockProductionLoop(ctx, bc, consensusEngine, privKey, validatorSet)
	}

	// Start consensus commit handler
	go handleCommits(ctx, bc, consensusEngine)

	// TODO: Start HTTP RPC server
	// TODO: Start gRPC server
	// TODO: Start metrics server

	fmt.Println("Node started successfully!")
	fmt.Println("Press Ctrl+C to stop...")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")
	return nil
}

func loadOrGenerateKey(keyFile string) (*crypto.PrivateKey, error) {
	if keyFile != "" {
		// Load from file
		data, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, err
		}
		return crypto.NewPrivateKeyFromBytes(data)
	}

	// Generate new key
	privKey, _, err := crypto.GenerateKey()
	return privKey, err
}

func createInitialValidatorSet(privKey *crypto.PrivateKey) *consensus.ValidatorSet {
	validators := make([]*consensus.Validator, 0)

	if privKey != nil {
		pubKey := privKey.Public()
		validator := consensus.NewValidator(
			pubKey.Address(),
			pubKey.Bytes(),
			1000000, // 1M voting power
		)
		validators = append(validators, validator)
	}

	// For demo, add some hardcoded validators if none provided
	if len(validators) == 0 {
		// Create dummy validator
		dummyPriv, dummyPub, _ := crypto.GenerateKey()
		_ = dummyPriv // Unused
		validator := consensus.NewValidator(
			dummyPub.Address(),
			dummyPub.Bytes(),
			1000000,
		)
		validators = append(validators, validator)
	}

	return consensus.NewValidatorSet(validators)
}

func registerP2PHandlers(network *p2p.Network, bc *blockchain.Blockchain, engine *consensus.Engine) {
	// Handle transaction messages
	network.RegisterHandler(p2p.MsgTypeTx, func(peerID peer.ID, msg *p2p.Message) error {
		tx, err := p2p.DecodeTxMessage(msg.Payload)
		if err != nil {
			return err
		}
		return bc.AddTransaction(tx)
	})

	// Handle block messages
	network.RegisterHandler(p2p.MsgTypeBlock, func(peerID peer.ID, msg *p2p.Message) error {
		block, err := p2p.DecodeBlockMessage(msg.Payload)
		if err != nil {
			return err
		}
		return engine.ProposeBlock(block)
	})

	// Handle vote messages
	network.RegisterHandler(p2p.MsgTypeVote, func(peerID peer.ID, msg *p2p.Message) error {
		vote, err := p2p.DecodeVoteMessage(msg.Payload)
		if err != nil {
			return err
		}
		return engine.AddVote(vote)
	})
}

func blockProductionLoop(ctx context.Context, bc *blockchain.Blockchain, engine *consensus.Engine, privKey *crypto.PrivateKey, validatorSet *consensus.ValidatorSet) {
	ticker := time.NewTicker(consensus.BlockTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check if we are the proposer
			height := bc.GetCurrentHeight() + 1
			proposer := validatorSet.GetProposer(height, []byte("seed"))

			if proposer != nil && proposer.Address == privKey.Public().Address() {
				// Propose a block
				block, err := bc.ProposeBlock(proposer.Address, privKey)
				if err != nil {
					fmt.Printf("Failed to propose block: %v\n", err)
					continue
				}

				fmt.Printf("Proposed block %d with %d transactions\n", block.Header.Height, len(block.Transactions))

				// Submit to consensus
				if err := engine.ProposeBlock(block); err != nil {
					fmt.Printf("Failed to submit block to consensus: %v\n", err)
				}
			}
		}
	}
}

func handleCommits(ctx context.Context, bc *blockchain.Blockchain, engine *consensus.Engine) {
	for {
		select {
		case <-ctx.Done():
			return
		case block := <-engine.CommitChan():
			if err := bc.ProcessBlock(block); err != nil {
				fmt.Printf("Failed to process block: %v\n", err)
			} else {
				fmt.Printf("Committed block %d (hash: %s)\n", block.Header.Height, block.Hash.String()[:8])
			}
		}
	}
}

