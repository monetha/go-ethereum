package ethereum

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var ether = big.NewInt(1000000000000000000) // 1 ether in wei

func TestTransferer_Transfer(t *testing.T) {
	// Generate a new random account and a funded simulator
	key, _ := crypto.GenerateKey()
	auth := bind.NewKeyedTransactor(key)

	alloc := core.GenesisAlloc{auth.From: {Balance: ether}}
	sim := backends.NewSimulatedBackend(alloc)
	sim.Commit()

	tr := Transferer{sim}

	key2, _ := crypto.GenerateKey()
	auth2 := bind.NewKeyedTransactor(key2)

	amount := new(big.Int).Div(ether, big.NewInt(10))
	to := auth2.From

	auth.Value = amount
	tx, err := tr.Transfer(auth, to, nil)
	if err != nil {
		t.Fatal(err)
	}
	sim.Commit()

	ctx := context.TODO()
	trr, err := sim.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		t.Fatal(err)
	}

	if trr.Status != types.ReceiptStatusSuccessful {
		t.Errorf("unexpected transaction status: %v", trr.Status)
	}

	toBalance, err := sim.BalanceAt(ctx, to, nil)
	if err != nil {
		t.Fatal(err)
	}

	if amount.Cmp(toBalance) != 0 {
		t.Errorf("expected balance after Transfer is %v, but got %v", amount, toBalance)
	}
}

func TestTransferer_SuggestGasLimit(t *testing.T) {
	// Generate a new random account and a funded simulator
	key, _ := crypto.GenerateKey()
	auth := bind.NewKeyedTransactor(key)

	alloc := core.GenesisAlloc{auth.From: {Balance: ether}}
	sim := backends.NewSimulatedBackend(alloc)
	sim.Commit()

	tr := Transferer{sim}

	key2, _ := crypto.GenerateKey()
	auth2 := bind.NewKeyedTransactor(key2)

	amount := new(big.Int).Div(ether, big.NewInt(10))
	to := auth2.From

	auth.Value = amount
	gasLimit, err := tr.SuggestGasLimit(auth, to, nil)
	if err != nil {
		t.Fatal(err)
	}

	expectedGasLimit := big.NewInt(21000)
	if expectedGasLimit.Cmp(gasLimit) != 0 {
		t.Errorf("expected gas limit is %v, but got %v", expectedGasLimit, gasLimit)
	}
}
