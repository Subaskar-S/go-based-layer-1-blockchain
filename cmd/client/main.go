package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/blockchain/layer1/internal/crypto"
	"github.com/blockchain/layer1/internal/types"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "layer1-cli",
		Short: "Layer-1 Blockchain CLI Client",
		Long:  "Command-line client for interacting with the Layer-1 blockchain",
	}

	// Key management commands
	keygenCmd := &cobra.Command{
		Use:   "keygen",
		Short: "Generate a new keypair",
		RunE:  runKeygen,
	}
	keygenCmd.Flags().String("output", "key.json", "Output file for the keypair")

	// Transaction commands
	transferCmd := &cobra.Command{
		Use:   "transfer",
		Short: "Create a transfer transaction",
		RunE:  runTransfer,
	}
	transferCmd.Flags().String("from", "", "Sender address")
	transferCmd.Flags().String("to", "", "Recipient address")
	transferCmd.Flags().Uint64("amount", 0, "Amount to transfer")
	transferCmd.Flags().Uint64("nonce", 0, "Transaction nonce")
	transferCmd.Flags().Uint64("gas-limit", 21000, "Gas limit")
	transferCmd.Flags().Uint64("gas-price", 1, "Gas price")
	transferCmd.Flags().String("key", "", "Private key file")
	transferCmd.MarkFlagRequired("from")
	transferCmd.MarkFlagRequired("to")
	transferCmd.MarkFlagRequired("amount")
	transferCmd.MarkFlagRequired("key")

	// Query commands
	balanceCmd := &cobra.Command{
		Use:   "balance",
		Short: "Query account balance",
		RunE:  runBalance,
	}
	balanceCmd.Flags().String("address", "", "Account address")
	balanceCmd.Flags().String("rpc", "http://localhost:8545", "RPC endpoint")
	balanceCmd.MarkFlagRequired("address")

	blockCmd := &cobra.Command{
		Use:   "block",
		Short: "Query block information",
		RunE:  runBlock,
	}
	blockCmd.Flags().Uint64("height", 0, "Block height")
	blockCmd.Flags().String("rpc", "http://localhost:8545", "RPC endpoint")

	rootCmd.AddCommand(keygenCmd, transferCmd, balanceCmd, blockCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runKeygen(cmd *cobra.Command, args []string) error {
	output, _ := cmd.Flags().GetString("output")

	// Generate keypair
	privKey, pubKey, err := crypto.GenerateKey()
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	address := pubKey.Address()

	// Create key data
	keyData := map[string]string{
		"private_key": hex.EncodeToString(privKey.Bytes()),
		"public_key":  hex.EncodeToString(pubKey.Bytes()),
		"address":     address.String(),
	}

	// Write to file
	data, err := json.MarshalIndent(keyData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal key data: %w", err)
	}

	if err := os.WriteFile(output, data, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	fmt.Printf("Generated new keypair:\n")
	fmt.Printf("Address: %s\n", address.String())
	fmt.Printf("Public Key: %s\n", hex.EncodeToString(pubKey.Bytes()))
	fmt.Printf("Private key saved to: %s\n", output)

	return nil
}

func runTransfer(cmd *cobra.Command, args []string) error {
	fromStr, _ := cmd.Flags().GetString("from")
	toStr, _ := cmd.Flags().GetString("to")
	amount, _ := cmd.Flags().GetUint64("amount")
	nonce, _ := cmd.Flags().GetUint64("nonce")
	gasLimit, _ := cmd.Flags().GetUint64("gas-limit")
	gasPrice, _ := cmd.Flags().GetUint64("gas-price")
	keyFile, _ := cmd.Flags().GetString("key")

	// Parse addresses
	from, err := crypto.AddressFromString(fromStr)
	if err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}

	to, err := crypto.AddressFromString(toStr)
	if err != nil {
		return fmt.Errorf("invalid to address: %w", err)
	}

	// Load private key
	keyData, err := os.ReadFile(keyFile)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	var keyJSON map[string]string
	if err := json.Unmarshal(keyData, &keyJSON); err != nil {
		return fmt.Errorf("failed to parse key file: %w", err)
	}

	privKeyBytes, err := hex.DecodeString(keyJSON["private_key"])
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	privKey, err := crypto.NewPrivateKeyFromBytes(privKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to load private key: %w", err)
	}

	// Create transaction
	tx := types.NewTransaction(
		types.TxTypeTransfer,
		from,
		to,
		nonce,
		amount,
		gasLimit,
		gasPrice,
		nil,
	)

	// Sign transaction
	if err := tx.Sign(privKey); err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Serialize transaction
	txData, err := tx.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize transaction: %w", err)
	}

	// Output transaction
	fmt.Printf("Transaction created:\n")
	fmt.Printf("Hash: %s\n", tx.Hash.String())
	fmt.Printf("From: %s\n", tx.From.String())
	fmt.Printf("To: %s\n", tx.To.String())
	fmt.Printf("Amount: %d\n", tx.Amount)
	fmt.Printf("Nonce: %d\n", tx.Nonce)
	fmt.Printf("Gas: %d @ %d\n", tx.GasLimit, tx.GasPrice)
	fmt.Printf("\nSerialized transaction (submit via RPC):\n%s\n", hex.EncodeToString(txData))

	// TODO: Submit to RPC endpoint

	return nil
}

func runBalance(cmd *cobra.Command, args []string) error {
	addressStr, _ := cmd.Flags().GetString("address")
	rpcEndpoint, _ := cmd.Flags().GetString("rpc")

	address, err := crypto.AddressFromString(addressStr)
	if err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	fmt.Printf("Querying balance for address: %s\n", address.String())
	fmt.Printf("RPC endpoint: %s\n", rpcEndpoint)

	// TODO: Query via RPC
	fmt.Println("Balance: 0 (RPC not implemented yet)")

	return nil
}

func runBlock(cmd *cobra.Command, args []string) error {
	height, _ := cmd.Flags().GetUint64("height")
	rpcEndpoint, _ := cmd.Flags().GetString("rpc")

	fmt.Printf("Querying block at height: %d\n", height)
	fmt.Printf("RPC endpoint: %s\n", rpcEndpoint)

	// TODO: Query via RPC
	fmt.Println("Block info: (RPC not implemented yet)")

	return nil
}

