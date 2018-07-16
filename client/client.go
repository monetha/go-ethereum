package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/monetha/go-ethereum"
)

// Client defines typed wrappers for the Ethereum RPC API.
type Client struct {
	c *rpc.Client
}

// Close implements io.Closer interface
func (c *Client) Close() error {
	c.c.Close()
	return nil
}

// Dial connects a client to the given URL.
func Dial(rawurl string) (*Client, error) {
	c, err := rpc.Dial(rawurl)
	if err != nil {
		return nil, err
	}
	return &Client{c}, nil
}

// BlockNumber returns the number of most recent block.
func (c *Client) BlockNumber(ctx context.Context) (*big.Int, error) {
	var number hexutil.Big
	err := c.c.CallContext(ctx, &number, "eth_blockNumber")
	if err != nil {
		return nil, fmt.Errorf("eth_blockNumber: %v", err)
	}

	return (*big.Int)(&number), nil
}

// BlockByNumber returns a block from the current canonical chain. If number is nil, the
// latest known block is returned.
func (c *Client) BlockByNumber(ctx context.Context, number *big.Int) (*ethereum.Block, error) {
	return c.getBlock(ctx, "eth_getBlockByNumber", toBlockNumArg(number), true)
}

func (c *Client) getBlock(ctx context.Context, method string, args ...interface{}) (*ethereum.Block, error) {
	var raw json.RawMessage
	err := c.c.CallContext(ctx, &raw, method, args...)
	if err != nil {
		return nil, err
	} else if len(raw) == 0 {
		return nil, ethereum.ErrNotFound
	}
	var header *types.Header
	if err := json.Unmarshal(raw, &header); err != nil {
		return nil, err
	}
	var body rpcBlock
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}

	txs := body.Transactions
	btxs := make(ethereum.Transactions, 0, len(txs))
	for _, tx := range txs {
		btx := &ethereum.Transaction{
			BlockNumber:      tx.BlockNumber,
			From:             tx.From,
			GasLimit:         tx.GasLimit,
			GasPrice:         tx.GasPrice,
			Hash:             tx.Hash,
			Input:            tx.Input,
			Nonce:            tx.Nonce,
			To:               tx.To,
			TransactionIndex: tx.TransactionIndex,
			Value:            tx.Value,
		}
		btxs = append(btxs, btx)
	}

	// Load transaction receipts
	txLen := len(btxs)
	if txLen > 0 {
		receipts := make([]*rpcReceipt, txLen)

		chunkOffset := 0 // offset of first element in current chunk
		for _, chunkTxs := range chunkTransactions(btxs, 500) {
			chunkLen := len(chunkTxs)

			reqs := make([]rpc.BatchElem, chunkLen)
			for i := range chunkTxs {
				globalIdx := chunkOffset + i

				reqs[i] = rpc.BatchElem{
					Method: "eth_getTransactionReceipt",
					Args:   []interface{}{btxs[globalIdx].Hash},
					Result: &receipts[globalIdx],
				}
			}

			// batch call
			if err := c.c.BatchCallContext(ctx, reqs); err != nil {
				return nil, fmt.Errorf("getting transaction receipts (offset: %d, len: %d): %v", chunkOffset, chunkLen, err)
			}

			// response validation
			for i, req := range reqs {
				globalIdx := chunkOffset + i

				if req.Error != nil {
					return nil, fmt.Errorf("request error for transaction %d of block %v: %v", globalIdx, header.Number, req.Error)
				}
				if receipts[globalIdx] == nil {
					return nil, fmt.Errorf("got null receipt for transaction %d of block %v", globalIdx, header.Number)
				}
			}

			chunkOffset += chunkLen
		}

		// assigning receipt values to transaction fields
		for i, rcpt := range receipts {
			btxs[i].GasUsed = rcpt.GasUsed
			if rcpt.Status != nil {
				btxs[i].Status = (*ethereum.TransactionStatus)(rcpt.Status)
			}
			if rcpt.ContractAddress != nil {
				btxs[i].ContractAddress = rcpt.ContractAddress
			}
		}
	}

	block := &ethereum.Block{
		Difficulty:   header.Difficulty,
		ExtraData:    header.Extra,
		GasLimit:     new(big.Int).SetUint64(header.GasLimit),
		GasUsed:      new(big.Int).SetUint64(header.GasUsed),
		Hash:         body.Hash,
		Miner:        header.Coinbase,
		Number:       header.Number,
		Timestamp:    header.Time,
		Transactions: btxs,
	}

	return block, nil
}

func chunkTransactions(txs ethereum.Transactions, chunkSize int) (chunks []ethereum.Transactions) {
	if chunkSize <= 0 {
		panic("chunk size must be positive number")
	}

	txsLen := len(txs)
	for i := 0; i < txsLen; i += chunkSize {
		end := i + chunkSize

		if end > txsLen {
			end = txsLen
		}

		chunks = append(chunks, txs[i:end])
	}

	return
}

type rpcBlock struct {
	Hash         common.Hash      `json:"hash"`
	Transactions []rpcTransaction `json:"transactions"`
	UncleHashes  []common.Hash    `json:"uncles"`
}

type rpcTransaction struct {
	BlockNumber      *big.Int
	From             common.Address
	GasLimit         *big.Int
	GasPrice         *big.Int
	Hash             common.Hash
	Input            []byte
	Nonce            uint64
	To               *common.Address // nil means contract creation
	TransactionIndex uint64
	Value            *big.Int
	V                *big.Int
	R                *big.Int
	S                *big.Int
}

func (t *rpcTransaction) UnmarshalJSON(input []byte) error {
	type tx struct {
		BlockNumber      *hexutil.Big    `json:"blockNumber"`
		From             *common.Address `json:"from"`
		GasLimit         *hexutil.Big    `json:"gas"`
		GasPrice         *hexutil.Big    `json:"gasPrice"`
		Hash             *common.Hash    `json:"hash"`
		Input            hexutil.Bytes   `json:"input"`
		Nonce            *hexutil.Uint64 `json:"nonce"`
		To               *common.Address `json:"to"`
		TransactionIndex *hexutil.Uint64 `json:"transactionIndex"`
		Value            *hexutil.Big    `json:"value"`
		V                *hexutil.Big    `json:"v"`
		R                *hexutil.Big    `json:"r"`
		S                *hexutil.Big    `json:"s"`
	}
	var dec tx
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}

	if dec.BlockNumber == nil {
		return errors.New("missing required field 'blockNumber'")
	}
	t.BlockNumber = (*big.Int)(dec.BlockNumber)

	if dec.From == nil {
		return errors.New("missing required field 'from'")
	}
	t.From = *dec.From

	if dec.GasLimit == nil {
		return errors.New("missing required field 'gas'")
	}
	t.GasLimit = (*big.Int)(dec.GasLimit)

	if dec.GasPrice == nil {
		return errors.New("missing required field 'gasPrice'")
	}
	t.GasPrice = (*big.Int)(dec.GasPrice)

	if dec.Hash == nil {
		return errors.New("missing required field 'hash'")
	}
	t.Hash = *dec.Hash

	if dec.Input == nil {
		return errors.New("missing required field 'input'")
	}
	t.Input = dec.Input

	if dec.Nonce == nil {
		return errors.New("missing required field 'nonce'")
	}
	t.Nonce = uint64(*dec.Nonce)

	if dec.To != nil {
		t.To = dec.To
	}

	if dec.TransactionIndex == nil {
		return errors.New("missing required field 'transactionIndex'")
	}
	t.TransactionIndex = uint64(*dec.TransactionIndex)

	if dec.Value == nil {
		return errors.New("missing required field 'value'")
	}
	t.Value = (*big.Int)(dec.Value)

	if dec.V == nil {
		return errors.New("missing required field 'v'")
	}
	t.V = (*big.Int)(dec.V)

	if dec.R == nil {
		return errors.New("missing required field 'r'")
	}
	t.R = (*big.Int)(dec.R)

	if dec.S == nil {
		return errors.New("missing required field 's'")
	}
	t.S = (*big.Int)(dec.S)

	return nil
}

type rpcReceipt struct {
	Status          *uint
	ContractAddress *common.Address
	GasUsed         *big.Int
}

func (r *rpcReceipt) UnmarshalJSON(input []byte) error {
	type Receipt struct {
		Status          *hexutil.Uint   `json:"status"`
		ContractAddress *common.Address `json:"contractAddress"`
		GasUsed         *hexutil.Big    `json:"gasUsed"`
	}
	var dec Receipt
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}

	if dec.Status != nil {
		r.Status = (*uint)(dec.Status)
	}

	if dec.ContractAddress != nil {
		r.ContractAddress = dec.ContractAddress
	}

	if dec.GasUsed == nil {
		return errors.New("missing required field 'gasUsed'")
	}
	r.GasUsed = (*big.Int)(dec.GasUsed)

	return nil
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	return hexutil.EncodeBig(number)
}
