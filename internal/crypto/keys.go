package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Address represents a blockchain address (20 bytes)
type Address [20]byte

// Hash represents a 32-byte hash
type Hash [32]byte

// Signature represents an Ed25519 signature
type Signature []byte

// PrivateKey wraps ed25519 private key
type PrivateKey struct {
	key ed25519.PrivateKey
}

// PublicKey wraps ed25519 public key
type PublicKey struct {
	key ed25519.PublicKey
}

// GenerateKey generates a new Ed25519 keypair
func GenerateKey() (*PrivateKey, *PublicKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate key: %w", err)
	}

	return &PrivateKey{key: priv}, &PublicKey{key: pub}, nil
}

// NewPrivateKeyFromBytes creates a private key from bytes
func NewPrivateKeyFromBytes(b []byte) (*PrivateKey, error) {
	if len(b) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: %d", len(b))
	}
	return &PrivateKey{key: ed25519.PrivateKey(b)}, nil
}

// NewPublicKeyFromBytes creates a public key from bytes
func NewPublicKeyFromBytes(b []byte) (*PublicKey, error) {
	if len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: %d", len(b))
	}
	return &PublicKey{key: ed25519.PublicKey(b)}, nil
}

// Bytes returns the raw private key bytes
func (pk *PrivateKey) Bytes() []byte {
	return pk.key
}

// Public returns the corresponding public key
func (pk *PrivateKey) Public() *PublicKey {
	return &PublicKey{key: pk.key.Public().(ed25519.PublicKey)}
}

// Sign signs a message with the private key
func (pk *PrivateKey) Sign(msg []byte) Signature {
	return ed25519.Sign(pk.key, msg)
}

// Bytes returns the raw public key bytes
func (pk *PublicKey) Bytes() []byte {
	return pk.key
}

// Address derives an address from the public key
// Address = first 20 bytes of SHA256(publicKey)
func (pk *PublicKey) Address() Address {
	hash := sha256.Sum256(pk.key)
	var addr Address
	copy(addr[:], hash[:20])
	return addr
}

// Verify verifies a signature against a message
func (pk *PublicKey) Verify(msg []byte, sig Signature) bool {
	return ed25519.Verify(pk.key, msg, sig)
}

// String returns hex representation of address
func (a Address) String() string {
	return hex.EncodeToString(a[:])
}

// Bytes returns address as byte slice
func (a Address) Bytes() []byte {
	return a[:]
}

// AddressFromBytes creates an address from bytes
func AddressFromBytes(b []byte) (Address, error) {
	if len(b) != 20 {
		return Address{}, fmt.Errorf("invalid address length: %d", len(b))
	}
	var addr Address
	copy(addr[:], b)
	return addr, nil
}

// AddressFromString creates an address from hex string
func AddressFromString(s string) (Address, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return Address{}, fmt.Errorf("invalid hex string: %w", err)
	}
	return AddressFromBytes(b)
}

// String returns hex representation of hash
func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

// Bytes returns hash as byte slice
func (h Hash) Bytes() []byte {
	return h[:]
}

// HashFromBytes creates a hash from bytes
func HashFromBytes(b []byte) (Hash, error) {
	if len(b) != 32 {
		return Hash{}, fmt.Errorf("invalid hash length: %d", len(b))
	}
	var hash Hash
	copy(hash[:], b)
	return hash, nil
}

// HashData computes SHA256 hash of data
func HashData(data []byte) Hash {
	return sha256.Sum256(data)
}

// ZeroAddress returns the zero address
func ZeroAddress() Address {
	return Address{}
}

// ZeroHash returns the zero hash
func ZeroHash() Hash {
	return Hash{}
}

// IsZero checks if address is zero
func (a Address) IsZero() bool {
	return a == ZeroAddress()
}

// IsZero checks if hash is zero
func (h Hash) IsZero() bool {
	return h == ZeroHash()
}

