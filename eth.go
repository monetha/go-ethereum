package ethereum

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/monetha/go-ethereum/backend"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/monetha/go-ethereum/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Eth simplifies some operations with the Ethereum network
type Eth struct {
	Backend           backend.Backend
	LogFun            log.Fun
	SuggestedGasPrice *big.Int
}

// New creates new instance of Eth
func New(b backend.Backend, lf log.Fun) *Eth {
	return &Eth{
		Backend: b,
		LogFun:  lf,
	}
}

// NewSession creates an instance of Sessionclear
func (e *Eth) NewSession(key *ecdsa.PrivateKey) *Session {
	transactOpts := bind.NewKeyedTransactor(key)
	transactOpts.GasPrice = e.SuggestedGasPrice
	return &Session{
		Eth:          e,
		TransactOpts: *transactOpts,
	}
}

// UpdateSuggestedGasPrice initializes suggested gas price from backend
func (e *Eth) UpdateSuggestedGasPrice(ctx context.Context) error {
	gasPrice, err := e.Backend.SuggestGasPrice(ctx)
	if err != nil {
		return err
	}
	e.SuggestedGasPrice = gasPrice
	return nil
}

// NewHandleNonceBackend returns new instance of Eth which internally handles nonce of the given addresses. It still calls PendingNonceAt of
// inner backend, but returns PendingNonceAt as a maximum of pending nonce in block-chain and internally stored nonce.
// It increments nonce for the given addresses after each successfully sent transaction (transaction may eventually
// fail in block-cain).
// Implementation is not thread-safe and should be used within one goroutine because otherwise invocations of
// PendingNonceAt and SendTransaction should be done atomically to have sequence of nonce without gaps (so that
// nonce would be equal to number of transactions sent).
func (e *Eth) NewHandleNonceBackend(handleAddresses []common.Address) *Eth {
	res := *e
	res.Backend = backend.NewHandleNonceBackend(res.Backend, handleAddresses)
	return &res
}

// WaitForTxReceipt waits until the transaction is successfully mined. It returns error if receipt status is not equal to `types.ReceiptStatusSuccessful`.
func (e *Eth) WaitForTxReceipt(ctx context.Context, txHash common.Hash) (tr *types.Receipt, err error) {
	b := e.Backend

	txHashStr := txHash.Hex()
	e.Log("Waiting for transaction", "hash", txHashStr)

	defer func() {
		if err != nil {
			err = fmt.Errorf("waiting for tx(%v): %v", txHashStr, err)
		}
	}()

	type commiter interface {
		Commit()
	}
	if sim, ok := b.(commiter); ok {
		sim.Commit()
		tr, err = e.onlySuccessfulReceipt(b.TransactionReceipt(ctx, txHash))
		return
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(4 * time.Second):
		}

		tr, err = e.onlySuccessfulReceipt(b.TransactionReceipt(ctx, txHash))
		if err == ethereum.NotFound {
			continue
		}
		return
	}
}

func (e *Eth) onlySuccessfulReceipt(tr *types.Receipt, err error) (*types.Receipt, error) {
	if err != nil {
		return nil, err
	}
	if tr.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("tx failed: %+v", tr)
	}
	e.Log("Transaction successfully mined", "tx_hash", tr.TxHash.Hex(), "cumulative_gas_used", tr.CumulativeGasUsed)
	return tr, nil
}

// Log writes message to the log with the context data
func (e *Eth) Log(msg string, ctx ...interface{}) {
	lf := e.LogFun
	if lf != nil {
		lf(msg, ctx...)
	}
}

// Session provides holds basic pre-configured parameters like backend, authorization, logging
type Session struct {
	*Eth
	TransactOpts bind.TransactOpts
}

// IsEnoughFunds retrieves current account balance and checks if it's enough funds given gas limit.
// SetGasPrice needs to be called with non-nil parameter before calling this method.
func (s *Session) IsEnoughFunds(ctx context.Context, gasLimit int64) (enough bool, minBalance *big.Int, err error) {
	gasPrice := s.TransactOpts.GasPrice
	if gasPrice == nil {
		panic("gas price must be non nil")
	}

	minBalance = new(big.Int).Mul(big.NewInt(gasLimit), gasPrice)

	s.Log("Getting balance", "address", s.TransactOpts.From.Hex())

	var balance *big.Int
	balance, err = s.Backend.BalanceAt(ctx, s.TransactOpts.From, nil)
	if err != nil {
		err = fmt.Errorf("backend BalanceAt(%v): %v", s.TransactOpts.From.Hex(), err)
		return
	}

	if balance.Cmp(minBalance) == -1 {
		return
	}

	enough = true
	return
}
