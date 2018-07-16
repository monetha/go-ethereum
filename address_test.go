package ethereum

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestNewAddressFromHex(t *testing.T) {
	testCases := map[string]struct {
		isValid  bool
		expected common.Address
	}{
		"0x":     {false, common.Address{}},
		"0x0":    {false, common.Address{}},
		"0x0000": {false, common.Address{}},
		"0x000000000000000000000000000000000000000":  {false, common.Address{}},
		"0xZ000000000000000000000000000000000000000": {false, common.Address{}},
		"0X0000000000000000000000000000000000000000": {true, common.Address{}},
		"0x0000000000000000000000000000000000000000": {true, common.Address{}},
		"0X00832A758A781055Ac19B5F9bF553Db8BB9db32D": {true,
			common.Address{0x00, 0x83, 0x2a, 0x75, 0x8a, 0x78, 0x10, 0x55, 0xac, 0x19, 0xb5, 0xf9, 0xbf, 0x55, 0x3d, 0xb8, 0xbb, 0x9d, 0xb3, 0x2d},
		},
		"0x00832A758A781055Ac19B5F9bF553Db8BB9db32D": {true,
			common.Address{0x00, 0x83, 0x2a, 0x75, 0x8a, 0x78, 0x10, 0x55, 0xac, 0x19, 0xb5, 0xf9, 0xbf, 0x55, 0x3d, 0xb8, 0xbb, 0x9d, 0xb3, 0x2d},
		},
		"0x00832a758a781055ac19b5f9bf553db8bb9db32d": {true,
			common.Address{0x00, 0x83, 0x2a, 0x75, 0x8a, 0x78, 0x10, 0x55, 0xac, 0x19, 0xb5, 0xf9, 0xbf, 0x55, 0x3d, 0xb8, 0xbb, 0x9d, 0xb3, 0x2d},
		},
		"00832A758A781055Ac19B5F9bF553Db8BB9db32D": {true,
			common.Address{0x00, 0x83, 0x2a, 0x75, 0x8a, 0x78, 0x10, 0x55, 0xac, 0x19, 0xb5, 0xf9, 0xbf, 0x55, 0x3d, 0xb8, 0xbb, 0x9d, 0xb3, 0x2d},
		},
	}
	for hex, tc := range testCases {
		t.Run(fmt.Sprintf("NewAddressFromHex(%s): is not valid = %v", hex, tc.isValid), func(t *testing.T) {
			addr, err := NewAddressFromHex(hex)
			if tc.isValid != (err == nil) {
				if err == nil {
					t.Errorf("Expected error")
				} else {
					t.Errorf("Got unexpected error: %v", err)
				}
			}

			if addr != tc.expected {
				t.Errorf("Expected address %+#v, but got: %+#v", tc.expected, addr)
			}
		})
	}
}
