package backend

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	handledAddressKey, _    = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	nonHandledAddressKey, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	handledAddress          = crypto.PubkeyToAddress(handledAddressKey.PublicKey)
	nonHandledAddress       = crypto.PubkeyToAddress(nonHandledAddressKey.PublicKey)
)

func TestHandleNonceBackend_PendingNonceAt(t *testing.T) {
	t.Run("returns error for any address when inner backend returns error", func(t *testing.T) {
		pendingNonceErr := errors.New("PendingNonceAt failed")
		inner := &backendMock{PendingNonceAtFunc: func(ctx context.Context, account common.Address) (uint64, error) {
			return 0, pendingNonceErr
		}}
		b := NewHandleNonceBackend(inner, []common.Address{handledAddress})
		ctx := context.TODO()

		_, err := b.PendingNonceAt(ctx, handledAddress)
		if err != pendingNonceErr {
			t.Errorf("expected error %v, but got %v", pendingNonceErr, err)
		}

		_, err = b.PendingNonceAt(ctx, nonHandledAddress)
		if err != pendingNonceErr {
			t.Errorf("expected error %v, but got %v", pendingNonceErr, err)
		}
	})

	t.Run("returns latest maximum value of pending nonce for handled addresses", func(t *testing.T) {
		maxNonce := uint64(12)
		var innerNonce uint64
		inner := &backendMock{PendingNonceAtFunc: func(ctx context.Context, account common.Address) (uint64, error) {
			return innerNonce, nil
		}}
		b := NewHandleNonceBackend(inner, []common.Address{handledAddress})
		ctx := context.TODO()

		innerNonce = maxNonce
		nonce, err := b.PendingNonceAt(ctx, handledAddress)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if nonce != innerNonce {
			t.Errorf("expected nonce %v, but got %v", innerNonce, nonce)
		}

		innerNonce = maxNonce - 1
		nonce, err = b.PendingNonceAt(ctx, handledAddress)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if nonce != maxNonce {
			t.Errorf("expected nonce %v, but got %v", innerNonce, nonce)
		}
	})

	t.Run("returns inner pending nonce for non-handled addresses", func(t *testing.T) {
		maxNonce := uint64(12)
		innerNonce := maxNonce
		inner := &backendMock{PendingNonceAtFunc: func(ctx context.Context, account common.Address) (uint64, error) {
			return innerNonce, nil
		}}
		b := NewHandleNonceBackend(inner, []common.Address{handledAddress})
		ctx := context.TODO()

		nonce, err := b.PendingNonceAt(ctx, nonHandledAddress)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if nonce != innerNonce {
			t.Errorf("expected nonce %v, but got %v", innerNonce, nonce)
		}

		innerNonce = maxNonce - 1
		nonce, err = b.PendingNonceAt(ctx, nonHandledAddress)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if nonce != innerNonce {
			t.Errorf("expected nonce %v, but got %v", innerNonce, nonce)
		}
	})
}

func TestHandleNonceBackend_SendTransaction(t *testing.T) {
	t.Run("increments pending nonce for handled address", func(t *testing.T) {
		innerNonce := uint64(12)
		sendTxNonce := uint64(50)
		inner := &backendMock{
			PendingNonceAtFunc: func(ctx context.Context, account common.Address) (uint64, error) {
				return innerNonce, nil
			},
			SendTransactionFunc: func(ctx context.Context, tx *types.Transaction) error {
				return nil
			},
		}
		b := NewHandleNonceBackend(inner, []common.Address{handledAddress})
		ctx := context.TODO()
		tx := createTx(handledAddressKey, sendTxNonce, nonHandledAddress)

		err := b.SendTransaction(ctx, tx)
		if err != nil {
			t.Fatalf("SendTransaction: %v", err)
		}

		nonce, err := b.PendingNonceAt(ctx, handledAddress)
		if err != nil {
			t.Fatalf("PendingNonceAt: %v", err)
		}

		if nonce != sendTxNonce+1 {
			t.Errorf("expected pending nonce after tx sent is %v, but got %v", sendTxNonce+1, nonce)
		}
	})

	t.Run("doesn't increment pending nonce for non-handled address", func(t *testing.T) {
		innerNonce := uint64(12)
		sendTxNonce := uint64(50)
		inner := &backendMock{
			PendingNonceAtFunc: func(ctx context.Context, account common.Address) (uint64, error) {
				return innerNonce, nil
			},
			SendTransactionFunc: func(ctx context.Context, tx *types.Transaction) error {
				return nil
			},
		}
		b := NewHandleNonceBackend(inner, []common.Address{handledAddress})
		ctx := context.TODO()
		tx := createTx(nonHandledAddressKey, sendTxNonce, handledAddress)

		err := b.SendTransaction(ctx, tx)
		if err != nil {
			t.Fatalf("SendTransaction: %v", err)
		}

		nonce, err := b.PendingNonceAt(ctx, nonHandledAddress)
		if err != nil {
			t.Fatalf("PendingNonceAt: %v", err)
		}

		if nonce != innerNonce {
			t.Errorf("expected pending nonce after tx sent is %v, but got %v", innerNonce, nonce)
		}
	})

	t.Run("doesn't increment pending nonce for handled address when transaction send failed", func(t *testing.T) {
		innerNonce := uint64(12)
		sendTxNonce := uint64(50)
		sendTransactionErr := errors.New("SendTransaction failed")
		inner := &backendMock{
			PendingNonceAtFunc: func(ctx context.Context, account common.Address) (uint64, error) {
				return innerNonce, nil
			},
			SendTransactionFunc: func(ctx context.Context, tx *types.Transaction) error {
				return sendTransactionErr
			},
		}
		b := NewHandleNonceBackend(inner, []common.Address{handledAddress})
		ctx := context.TODO()
		tx := createTx(handledAddressKey, sendTxNonce, nonHandledAddress)

		err := b.SendTransaction(ctx, tx)
		if err != sendTransactionErr {
			t.Errorf("expected error %v, but got %v", sendTransactionErr, err)
		}

		nonce, err := b.PendingNonceAt(ctx, handledAddress)
		if err != nil {
			t.Fatalf("PendingNonceAt: %v", err)
		}

		if nonce != innerNonce {
			t.Errorf("expected pending nonce after tx sent is %v, but got %v", sendTxNonce+1, nonce)
		}
	})
}

func createTx(key *ecdsa.PrivateKey, nonce uint64, to common.Address) *types.Transaction {
	opts := bind.NewKeyedTransactor(key)
	opts.Value = big.NewInt(1000000000000000000)
	opts.GasLimit = uint64(21000)
	opts.GasPrice = big.NewInt(1000000000)
	rawTx := types.NewTransaction(nonce, to, opts.Value, opts.GasLimit, opts.GasPrice, nil)
	tx, err := opts.Signer(types.HomesteadSigner{}, opts.From, rawTx)
	if err != nil {
		panic(err)
	}
	return tx
}

type backendMock struct {
	CodeAtFunc             func(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error)
	CallContractFunc       func(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
	PendingCodeAtFunc      func(ctx context.Context, account common.Address) ([]byte, error)
	PendingNonceAtFunc     func(ctx context.Context, account common.Address) (uint64, error)
	SuggestGasPriceFunc    func(ctx context.Context) (*big.Int, error)
	EstimateGasFunc        func(ctx context.Context, call ethereum.CallMsg) (usedGas uint64, err error)
	SendTransactionFunc    func(ctx context.Context, tx *types.Transaction) error
	TransactionReceiptFunc func(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	BalanceAtFunc          func(ctx context.Context, address common.Address, blockNum *big.Int) (*big.Int, error)
}

func (m *backendMock) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	return m.CodeAtFunc(ctx, contract, blockNumber)
}

func (m *backendMock) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return m.CallContractFunc(ctx, call, blockNumber)
}

func (m *backendMock) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	return m.PendingCodeAtFunc(ctx, account)
}

func (m *backendMock) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return m.PendingNonceAtFunc(ctx, account)
}

func (m *backendMock) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return m.SuggestGasPriceFunc(ctx)
}

func (m *backendMock) EstimateGas(ctx context.Context, call ethereum.CallMsg) (usedGas uint64, err error) {
	return m.EstimateGasFunc(ctx, call)
}

func (m *backendMock) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return m.SendTransactionFunc(ctx, tx)
}

func (m *backendMock) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return m.TransactionReceiptFunc(ctx, txHash)
}

func (m *backendMock) BalanceAt(ctx context.Context, address common.Address, blockNum *big.Int) (*big.Int, error) {
	return m.BalanceAtFunc(ctx, address, blockNum)
}

func (m *backendMock) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	return nil, nil
}

func (m *backendMock) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	return nil, nil
}

func (m *backendMock) TransactionByHash(ctx context.Context, txHash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	return nil, false, nil
}
