package ethereum

import (
	"fmt"
	"testing"
)

func TestNewKeyFromPrivateKey(t *testing.T) {
	testCases := map[string]struct {
		isValid   bool
		address   string
		publicKey string
	}{
		"0000000000000000000000000000000000000000000000000000000000000001": {true, "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf",
			"79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8",
		},
		"0000000000000000000000000000000000000000000000000000000000000000": {false, "",
			"",
		},
		"aa22b54c0cb43ee30a014afe5ef3664b1cde299feabca46cd3167a85a57c39f2": {true, "0x006E27B6A72E1f34C626762F3C4761547Aff1421",
			"c4c5398da6843632c123f543d714d2d2277716c11ff612b2a2f23c6bda4d6f0327c31cd58c55a9572c3cc141dade0c32747a13b7ef34c241b26c84adbb28fcf4",
		},
	}

	for hexKey, expected := range testCases {
		t.Run(fmt.Sprintf("NewKeyFromPrivateKey(%s)", hexKey), func(t *testing.T) {
			k, err := NewKeyFromPrivateKey(hexKey)
			if expected.isValid != (err == nil) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				} else {
					t.Fatal("expected error")
				}
			}

			if err == nil {
				addr := k.AddressString()
				if addr != expected.address {
					t.Errorf("Expected address %v, but got: %v", expected.address, addr)
				}

				pubKey := k.PublicKeyString()
				if pubKey != expected.publicKey {
					t.Errorf("Expected public key %v, but got: %v", expected.publicKey, pubKey)
				}
			}
		})
	}
}
