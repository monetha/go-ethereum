package backend

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Backend contains all methods required for the backend operations.
type Backend interface {
	bind.ContractBackend
	ethereum.TransactionReader
	BalanceAt(ctx context.Context, address common.Address, blockNum *big.Int) (*big.Int, error)
}

// HandleNonceBackend internally handles nonce of the given addresses. It still calls PendingNonceAt of
// inner backend, but returns PendingNonceAt as a maximum of pending nonce in block-chain and internally stored nonce.
// It increments nonce for the given addresses after each successfully sent transaction (transaction may eventually
// fail in block-cain).
// Implementation is not thread-safe and should be used within one goroutine because otherwise invocations of
// PendingNonceAt and SendTransaction should be done atomically to have sequence of nonce without gaps (so that
// nonce would be equal to number of transactions sent).
type HandleNonceBackend struct {
	inner        Backend
	addressNonce map[common.Address]uint64
}

// NewHandleNonceBackend wraps backend and returns new instance of HandleNonceBackend.
func NewHandleNonceBackend(inner Backend, handleAddresses []common.Address) Backend {
	addressNonce := make(map[common.Address]uint64, len(handleAddresses))
	for _, address := range handleAddresses {
		addressNonce[address] = uint64(0)
	}
	b := &HandleNonceBackend{inner: inner, addressNonce: addressNonce}

	if cr, ok := b.inner.(commiterRollbacker); ok {
		return &simBackend{
			b:  b,
			cr: cr,
		}
	}

	return b
}

// CodeAt returns the code of the given account. This is needed to differentiate
// between contract internal errors and the local chain being out of sync.
func (b *HandleNonceBackend) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	return b.inner.CodeAt(ctx, contract, blockNumber)
}

// CallContract executes an Ethereum contract call with the specified data as the input.
func (b *HandleNonceBackend) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return b.inner.CallContract(ctx, call, blockNumber)
}

// PendingCodeAt returns the code of the given account in the pending state.
func (b *HandleNonceBackend) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	return b.inner.PendingCodeAt(ctx, account)
}

// PendingNonceAt retrieves the current pending nonce associated with an account.
func (b *HandleNonceBackend) PendingNonceAt(ctx context.Context, account common.Address) (nonce uint64, err error) {
	nonce, err = b.inner.PendingNonceAt(ctx, account)
	if err != nil {
		return
	}

	innerNonce, shouldHandle := b.addressNonce[account]
	if !shouldHandle {
		return
	}

	if nonce > innerNonce {
		b.addressNonce[account] = nonce
	} else {
		nonce = innerNonce
	}

	return
}

// SuggestGasPrice retrieves the currently suggested gas price to allow a timely
// execution of a transaction.
func (b *HandleNonceBackend) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return b.inner.SuggestGasPrice(ctx)
}

// EstimateGas tries to estimate the gas needed to execute a specific
// transaction based on the current pending state of the backend blockchain.
// There is no guarantee that this is the true gas limit requirement as other
// transactions may be added or removed by miners, but it should provide a basis
// for setting a reasonable default.
func (b *HandleNonceBackend) EstimateGas(ctx context.Context, call ethereum.CallMsg) (usedGas uint64, err error) {
	return b.inner.EstimateGas(ctx, call)
}

// SendTransaction injects the transaction into the pending pool for execution.
func (b *HandleNonceBackend) SendTransaction(ctx context.Context, tx *types.Transaction) (err error) {
	err = b.inner.SendTransaction(ctx, tx)
	if err != nil {
		return
	}

	b.incrementNonce(tx)

	return
}

func (b *HandleNonceBackend) incrementNonce(tx *types.Transaction) {
	from, err := types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)
	if err != nil {
		return // invalid sender
	}

	innerNonce, shouldHandle := b.addressNonce[from]
	if !shouldHandle {
		return
	}

	nonce := tx.Nonce() + 1
	if nonce > innerNonce {
		b.addressNonce[from] = nonce
	}
}

// TransactionReceipt returns the receipt of a transaction by transaction hash.
func (b *HandleNonceBackend) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return b.inner.TransactionReceipt(ctx, txHash)
}

// BalanceAt returns the balance of the account of given address.
func (b *HandleNonceBackend) BalanceAt(ctx context.Context, address common.Address, blockNum *big.Int) (*big.Int, error) {
	return b.inner.BalanceAt(ctx, address, blockNum)
}

// FilterLogs executes a log filter operation, blocking during execution and
// returning all the results in one batch.
func (b *HandleNonceBackend) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	return b.inner.FilterLogs(ctx, query)
}

// SubscribeFilterLogs creates a background log filtering operation, returning
// a subscription immediately, which can be used to stream the found events.
func (b *HandleNonceBackend) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	return b.inner.SubscribeFilterLogs(ctx, query, ch)
}

// TransactionByHash checks the pool of pending transactions in addition to the
// blockchain. The isPending return value indicates whether the transaction has been
// mined yet. Note that the transaction may not be part of the canonical chain even if
// it's not pending.
func (b *HandleNonceBackend) TransactionByHash(ctx context.Context, txHash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	return b.inner.TransactionByHash(ctx, txHash)
}

type commiterRollbacker interface {
	Commit()
	Rollback()
}

type simBackend struct {
	b  Backend
	cr commiterRollbacker
}

func (b *simBackend) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	return b.b.CodeAt(ctx, contract, blockNumber)
}

func (b *simBackend) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return b.b.CallContract(ctx, call, blockNumber)
}

func (b *simBackend) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	return b.b.PendingCodeAt(ctx, account)
}

func (b *simBackend) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return b.b.PendingNonceAt(ctx, account)
}

func (b *simBackend) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return b.b.SuggestGasPrice(ctx)
}

func (b *simBackend) EstimateGas(ctx context.Context, call ethereum.CallMsg) (usedGas uint64, err error) {
	return b.b.EstimateGas(ctx, call)
}

func (b *simBackend) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return b.b.SendTransaction(ctx, tx)
}

func (b *simBackend) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return b.b.TransactionReceipt(ctx, txHash)
}

func (b *simBackend) BalanceAt(ctx context.Context, address common.Address, blockNum *big.Int) (*big.Int, error) {
	return b.b.BalanceAt(ctx, address, blockNum)
}

func (b *simBackend) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	return b.b.FilterLogs(ctx, query)
}

func (b *simBackend) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	return b.b.SubscribeFilterLogs(ctx, query, ch)
}

func (b *simBackend) TransactionByHash(ctx context.Context, txHash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	return b.b.TransactionByHash(ctx, txHash)
}

func (b *simBackend) Commit() {
	b.cr.Commit()
}

func (b *simBackend) Rollback() {
	b.cr.Rollback()
}
