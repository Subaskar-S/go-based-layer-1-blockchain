package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKey(t *testing.T) {
	privKey, pubKey, err := GenerateKey()
	require.NoError(t, err)
	require.NotNil(t, privKey)
	require.NotNil(t, pubKey)

	// Verify key sizes
	assert.Equal(t, 64, len(privKey.Bytes()))
	assert.Equal(t, 32, len(pubKey.Bytes()))
}

func TestSignAndVerify(t *testing.T) {
	privKey, pubKey, err := GenerateKey()
	require.NoError(t, err)

	message := []byte("test message")
	signature := privKey.Sign(message)

	// Verify with correct public key
	assert.True(t, pubKey.Verify(message, signature))

	// Verify with different message should fail
	assert.False(t, pubKey.Verify([]byte("different message"), signature))

	// Verify with different public key should fail
	_, otherPubKey, _ := GenerateKey()
	assert.False(t, otherPubKey.Verify(message, signature))
}

func TestAddressDerivation(t *testing.T) {
	privKey, pubKey, err := GenerateKey()
	require.NoError(t, err)

	addr1 := pubKey.Address()
	addr2 := pubKey.Address()

	// Same public key should produce same address
	assert.Equal(t, addr1, addr2)

	// Address should be 20 bytes
	assert.Equal(t, 20, len(addr1.Bytes()))

	// Different public key should produce different address
	_, otherPubKey, _ := GenerateKey()
	otherAddr := otherPubKey.Address()
	assert.NotEqual(t, addr1, otherAddr)
}

func TestAddressFromString(t *testing.T) {
	_, pubKey, err := GenerateKey()
	require.NoError(t, err)

	addr := pubKey.Address()
	addrStr := addr.String()

	// Parse address from string
	parsedAddr, err := AddressFromString(addrStr)
	require.NoError(t, err)
	assert.Equal(t, addr, parsedAddr)

	// Invalid hex string should fail
	_, err = AddressFromString("invalid")
	assert.Error(t, err)

	// Wrong length should fail
	_, err = AddressFromString("0102030405")
	assert.Error(t, err)
}

func TestHashData(t *testing.T) {
	data := []byte("test data")
	hash1 := HashData(data)
	hash2 := HashData(data)

	// Same data should produce same hash
	assert.Equal(t, hash1, hash2)

	// Hash should be 32 bytes
	assert.Equal(t, 32, len(hash1.Bytes()))

	// Different data should produce different hash
	hash3 := HashData([]byte("different data"))
	assert.NotEqual(t, hash1, hash3)
}

func TestZeroAddress(t *testing.T) {
	zero := ZeroAddress()
	assert.True(t, zero.IsZero())

	_, pubKey, _ := GenerateKey()
	addr := pubKey.Address()
	assert.False(t, addr.IsZero())
}

func TestPrivateKeyFromBytes(t *testing.T) {
	privKey, _, err := GenerateKey()
	require.NoError(t, err)

	bytes := privKey.Bytes()
	restored, err := NewPrivateKeyFromBytes(bytes)
	require.NoError(t, err)

	// Restored key should be identical
	assert.Equal(t, privKey.Bytes(), restored.Bytes())

	// Invalid size should fail
	_, err = NewPrivateKeyFromBytes([]byte{1, 2, 3})
	assert.Error(t, err)
}

func TestPublicKeyFromBytes(t *testing.T) {
	_, pubKey, err := GenerateKey()
	require.NoError(t, err)

	bytes := pubKey.Bytes()
	restored, err := NewPublicKeyFromBytes(bytes)
	require.NoError(t, err)

	// Restored key should be identical
	assert.Equal(t, pubKey.Bytes(), restored.Bytes())
	assert.Equal(t, pubKey.Address(), restored.Address())

	// Invalid size should fail
	_, err = NewPublicKeyFromBytes([]byte{1, 2, 3})
	assert.Error(t, err)
}

func BenchmarkGenerateKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateKey()
	}
}

func BenchmarkSign(b *testing.B) {
	privKey, _, _ := GenerateKey()
	message := []byte("benchmark message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		privKey.Sign(message)
	}
}

func BenchmarkVerify(b *testing.B) {
	privKey, pubKey, _ := GenerateKey()
	message := []byte("benchmark message")
	signature := privKey.Sign(message)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pubKey.Verify(message, signature)
	}
}

func BenchmarkHashData(b *testing.B) {
	data := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HashData(data)
	}
}

