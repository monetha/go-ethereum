package ethereum

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Key is combination of public address and private key.
type Key struct {
	// Address is address associated with the PrivateKey (stored for simplicity)
	Address common.Address
	// PrivateKey represents a ECDSA private key. Pubkey/address can be derived from it.
	PrivateKey *ecdsa.PrivateKey
}

// NewKey generates a random Key.
func NewKey() (*Key, error) {
	return newKey(rand.Reader)
}

// NewKeyFromPrivateKey creates Key from hex-string representation of private key.
func NewKeyFromPrivateKey(hexkey string) (*Key, error) {
	privateKeyECDSA, err := crypto.HexToECDSA(hexkey)
	if err != nil {
		return nil, err
	}
	return newKeyFromECDSA(privateKeyECDSA)
}

// PrivateKeyString returns hex-string representation of private key.
func (k *Key) PrivateKeyString() string {
	return hex.EncodeToString(crypto.FromECDSA(k.PrivateKey))
}

// PublicKeyString returns hex-string representation of public key.
func (k *Key) PublicKeyString() string {
	return hex.EncodeToString(crypto.FromECDSAPub(&k.PrivateKey.PublicKey)[1:])
}

// AddressString returns string representation of address.
func (k *Key) AddressString() string {
	return k.Address.String()
}

func newKey(rand io.Reader) (*Key, error) {
	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), rand)
	if err != nil {
		return nil, err
	}
	return newKeyFromECDSA(privateKeyECDSA)
}

func newKeyFromECDSA(privateKeyECDSA *ecdsa.PrivateKey) (*Key, error) {
	address, err := pubkeyToAddress(privateKeyECDSA.PublicKey)
	if err != nil {
		return nil, err
	}

	return &Key{
		Address:    address,
		PrivateKey: privateKeyECDSA,
	}, nil
}

func pubkeyToAddress(p ecdsa.PublicKey) (common.Address, error) {
	pubBytes := crypto.FromECDSAPub(&p)
	if pubBytes == nil {
		return common.Address{}, errors.New("invalid key")
	}
	return common.Address(common.BytesToAddress(crypto.Keccak256(pubBytes[1:])[12:])), nil
}
