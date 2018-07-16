package ethereum

import (
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// NewAddressFromHex parses hex string and returns address.
func NewAddressFromHex(hex string) (addr common.Address, err error) {
	if !isHexAddress(hex) {
		err = fmt.Errorf("ethereum: hex string '%v' is not valid Ethereum address", hex)
		return
	}

	var bs []byte
	bs, err = fromHex(hex)
	if err == nil {
		addr = common.BytesToAddress(bs)
	}

	return
}

func isHexAddress(s string) bool {
	if len(s) == 2+2*common.AddressLength && isHex(s) {
		return true
	}
	if len(s) == 2*common.AddressLength && isHex("0x"+s) {
		return true
	}
	return false
}

func isHex(str string) bool {
	l := len(str)
	return l >= 4 && l%2 == 0 && (str[0:2] == "0x" || str[0:2] == "0X")
}

func fromHex(s string) ([]byte, error) {
	if len(s) > 1 {
		if s[0:2] == "0x" || s[0:2] == "0X" {
			s = s[2:]
		}
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	return hex.DecodeString(s)
}
