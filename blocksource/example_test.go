package blocksource_test

import (
	"fmt"
	"math/big"

	"github.com/monetha/go-ethereum/blocksource"
)

func ExampleNew() {
	config := &blocksource.Config{StartBlock: big.NewInt(4760755), Confirmations: 1}
	source, err := blocksource.New("https://mainnet.infura.io/", config)
	if err != nil {
		panic(err)
	}

	i := 0
	for b := range source.Blocks() {
		i++
		fmt.Printf("New block arrived: %v\n", b.Number)
		if i == 10 {
			source.Close()
		}
	}
}
