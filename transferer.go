package ethereum

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Transferer allows to make ethers transfer between accounts.
type Transferer struct {
	ContractTransactor bind.ContractTransactor
}

// SuggestGasLimit returns suggested gas limit to make transfer.
func (t Transferer) SuggestGasLimit(opts *bind.TransactOpts, to common.Address, input []byte) (gasLimit *big.Int, err error) {
	ct := t.ContractTransactor

	// Ensure a valid value field and resolve the account nonce
	value := opts.Value
	if value == nil {
		value = new(big.Int)
	}

	// estimate the transaction
	msg := ethereum.CallMsg{From: opts.From, To: &to, Value: value, Data: input}
	var gl uint64
	gl, err = ct.EstimateGas(ensureContext(opts.Context), msg)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate gas needed: %v", err)
	}
	gasLimit = new(big.Int).SetUint64(gl)
	return
}

// Transfer transfers ethers to `to` account. `input` is optional and can be set to nil.
func (t Transferer) Transfer(opts *bind.TransactOpts, to common.Address, input []byte) (*types.Transaction, error) {
	ct := t.ContractTransactor

	var err error
	// Ensure a valid value field and resolve the account nonce
	value := opts.Value
	if value == nil {
		value = new(big.Int)
	}
	var nonce uint64
	if opts.Nonce == nil {
		nonce, err = ct.PendingNonceAt(ensureContext(opts.Context), opts.From)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve account nonce: %v", err)
		}
	} else {
		nonce = opts.Nonce.Uint64()
	}
	// Figure out the gas allowance and gas price values
	gasPrice := opts.GasPrice
	if gasPrice == nil {
		gasPrice, err = ct.SuggestGasPrice(ensureContext(opts.Context))
		if err != nil {
			return nil, fmt.Errorf("failed to suggest gas price: %v", err)
		}
	}
	gasLimit := opts.GasLimit
	if gasLimit == 0 {
		// estimate the transaction
		msg := ethereum.CallMsg{From: opts.From, To: &to, Value: value, Data: input}
		gasLimit, err = ct.EstimateGas(ensureContext(opts.Context), msg)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate gas needed: %v", err)
		}
	}
	// Create the transaction, sign it and schedule it for execution
	rawTx := types.NewTransaction(nonce, to, value, gasLimit, gasPrice, input)
	if opts.Signer == nil {
		return nil, errors.New("no signer to authorize the transaction with")
	}
	signedTx, err := opts.Signer(types.HomesteadSigner{}, opts.From, rawTx)
	if err != nil {
		return nil, err
	}
	if err := ct.SendTransaction(ensureContext(opts.Context), signedTx); err != nil {
		return nil, err
	}
	return signedTx, nil
}

func ensureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.TODO()
	}
	return ctx
}
