package gasestimator

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// GasLimitEstimator is the gas estimator for the contract methods on the Ethereum network.
type GasLimitEstimator struct {
	address    common.Address          // Deployment address of the contract on the Ethereum blockchain
	abi        abi.ABI                 // Reflect based ABI to access the correct Ethereum methods
	transactor bind.ContractTransactor // Write interface to interact with the blockchain
}

// NewGasLimitEstimator creates instance of GasLimitEstimator
func NewGasLimitEstimator(address common.Address, abi abi.ABI, transactor bind.ContractTransactor) *GasLimitEstimator {
	return &GasLimitEstimator{address: address, abi: abi, transactor: transactor}
}

// EstimateGas estimates gas limit to call the (paid) contract method with params as input values.
func (c *GasLimitEstimator) EstimateGas(opts *bind.TransactOpts, method string, params ...interface{}) (*big.Int, error) {
	// Otherwise pack up the parameters and invoke the contract
	input, err := c.abi.Pack(method, params...)
	if err != nil {
		return nil, err
	}
	return c.estimateGas(opts, &c.address, input)
}

func (c *GasLimitEstimator) estimateGas(opts *bind.TransactOpts, contract *common.Address, input []byte) (*big.Int, error) {
	value := opts.Value
	if value == nil {
		value = new(big.Int)
	}

	// Gas estimation cannot succeed without code for method invocations
	if contract != nil {
		if code, err := c.transactor.PendingCodeAt(ensureContext(opts.Context), c.address); err != nil {
			return nil, err
		} else if len(code) == 0 {
			return nil, bind.ErrNoCode
		}
	}

	// If the contract surely has code (or code is not needed), estimate the transaction
	msg := ethereum.CallMsg{From: opts.From, To: contract, Value: value, Data: input}
	gasLimit, err := c.transactor.EstimateGas(ensureContext(opts.Context), msg)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate gas needed: %v", err)
	}

	return new(big.Int).SetUint64(gasLimit), nil
}

func ensureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.TODO()
	}
	return ctx
}
