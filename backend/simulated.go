// +build !js

package backend

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

// make sure SimulatedBackendExt implements Backend
var _ Backend = &SimulatedBackendExt{}

// NewSimulatedBackendExtended creates a new binding backend using a simulated blockchain
// for testing purposes. It uses `backends.SimulatedBackend` under the hood, but extends it to support
// `ethereum.TransactionReader` interface.
func NewSimulatedBackendExtended(alloc core.GenesisAlloc, gasLimit uint64) *SimulatedBackendExt {
	return &SimulatedBackendExt{
		b: backends.NewSimulatedBackend(alloc, 10000000),
	}
}

// SimulatedBackendExt wraps `backends.SimulatedBackend` and implements additionally `ethereum.TransactionReader` interface.
type SimulatedBackendExt struct {
	b   *backends.SimulatedBackend
	txs sync.Map
}

// CodeAt returns the code associated with a certain account in the blockchain.
func (b *SimulatedBackendExt) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	return b.b.CodeAt(ctx, contract, blockNumber)
}

// CallContract executes a contract call.
func (b *SimulatedBackendExt) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return b.b.CallContract(ctx, call, blockNumber)
}

// PendingCodeAt returns the code associated with an account in the pending state.
func (b *SimulatedBackendExt) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	return b.b.PendingCodeAt(ctx, account)
}

// PendingNonceAt implements PendingStateReader.PendingNonceAt, retrieving
// the nonce currently pending for the account.
func (b *SimulatedBackendExt) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return b.b.PendingNonceAt(ctx, account)
}

// SuggestGasPrice implements ContractTransactor.SuggestGasPrice. Since the simulated
// chain doens't have miners, we just return a gas price of 1 for any call.
func (b *SimulatedBackendExt) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return b.b.SuggestGasPrice(ctx)
}

// EstimateGas executes the requested code against the currently pending block/state and
// returns the used amount of gas.
func (b *SimulatedBackendExt) EstimateGas(ctx context.Context, call ethereum.CallMsg) (usedGas uint64, err error) {
	return b.b.EstimateGas(ctx, call)
}

// SendTransaction updates the pending block to include the given transaction.
// It panics if the transaction is invalid.
func (b *SimulatedBackendExt) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	err := b.b.SendTransaction(ctx, tx)
	if err == nil {
		b.txs.Store(tx.Hash(), tx)
	}

	return err
}

// TransactionReceipt returns the receipt of a transaction.
func (b *SimulatedBackendExt) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return b.b.TransactionReceipt(ctx, txHash)
}

// BalanceAt returns the wei balance of a certain account in the blockchain.
func (b *SimulatedBackendExt) BalanceAt(ctx context.Context, address common.Address, blockNum *big.Int) (*big.Int, error) {
	return b.b.BalanceAt(ctx, address, blockNum)
}

// FilterLogs executes a log filter operation, blocking during execution and
// returning all the results in one batch.
func (b *SimulatedBackendExt) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	return b.b.FilterLogs(ctx, query)
}

// SubscribeFilterLogs creates a background log filtering operation, returning a
// subscription immediately, which can be used to stream the found events.
func (b *SimulatedBackendExt) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	return b.b.SubscribeFilterLogs(ctx, query, ch)
}

// Commit imports all the pending transactions as a single block and starts a
// fresh new state.
func (b *SimulatedBackendExt) Commit() {
	b.b.Commit()
}

// Rollback aborts all pending transactions, reverting to the last committed state.
func (b *SimulatedBackendExt) Rollback() {
	b.b.Rollback()
}

// TransactionByHash checks the pool of pending transactions in addition to the
// blockchain. The isPending return value indicates whether the transaction has been
// mined yet. Note that the transaction may not be part of the canonical chain even if
// it's not pending.
func (b *SimulatedBackendExt) TransactionByHash(ctx context.Context, txHash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	v, ok := b.txs.Load(txHash)
	if !ok {
		return nil, false, ethereum.NotFound
	}
	tx = v.(*types.Transaction)

	txr, _ := b.b.TransactionReceipt(ctx, txHash)
	isPending = txr == nil

	return
}
