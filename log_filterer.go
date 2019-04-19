package ethereum

import (
	"context"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
)

// ContractLogFilterer extends FilterLogs method of bind.BoundContract struct to allow filter multiple events
type ContractLogFilterer struct {
	address  common.Address        // Deployment address of the contract on the Ethereum blockchain
	abi      abi.ABI               // Reflect based ABI to access the correct Ethereum methods
	filterer bind.ContractFilterer // Event filtering to interact with the blockchain
}

// NewContractLogFilterer creates a instance of ContractLogFilterer.
func NewContractLogFilterer(address common.Address, abi abi.ABI, filterer bind.ContractFilterer) *ContractLogFilterer {
	return &ContractLogFilterer{
		address:  address,
		abi:      abi,
		filterer: filterer,
	}
}

// FilterLogs filters contract logs for past blocks, returning the necessary
// channels to construct a strongly typed bound iterator on top of them.
func (c *ContractLogFilterer) FilterLogs(opts *bind.FilterOpts, names []string, query ...[]interface{}) (chan types.Log, event.Subscription, error) {
	// Don't crash on a lazy user
	if opts == nil {
		opts = new(bind.FilterOpts)
	}

	var eventNameRule []interface{}
	for _, name := range names {
		eventNameRule = append(eventNameRule, c.abi.Events[name].Id())
	}

	// Append the event selector to the query parameters and construct the topic set
	query = append([][]interface{}{eventNameRule}, query...)

	topics, err := makeTopics(query...)
	if err != nil {
		return nil, nil, err
	}
	// Start the background filtering
	logs := make(chan types.Log, 128)

	config := ethereum.FilterQuery{
		Addresses: []common.Address{c.address},
		Topics:    topics,
		FromBlock: new(big.Int).SetUint64(opts.Start),
	}
	if opts.End != nil {
		config.ToBlock = new(big.Int).SetUint64(*opts.End)
	}

	buff, err := c.filterer.FilterLogs(ensureContext(opts.Context), config)
	if err != nil {
		return nil, nil, err
	}
	sub, err := event.NewSubscription(func(quit <-chan struct{}) error {
		for _, log := range buff {
			select {
			case logs <- log:
			case <-quit:
				return nil
			}
		}
		return nil
	}), nil

	if err != nil {
		return nil, nil, err
	}
	return logs, sub, nil
}

// makeTopics converts a filter query argument list into a filter topic set.
func makeTopics(query ...[]interface{}) ([][]common.Hash, error) {
	topics := make([][]common.Hash, len(query))
	for i, filter := range query {
		for _, rule := range filter {
			var topic common.Hash

			// Try to generate the topic based on simple types
			switch rule := rule.(type) {
			case common.Hash:
				copy(topic[:], rule[:])
			case common.Address:
				copy(topic[common.HashLength-common.AddressLength:], rule[:])
			case *big.Int:
				blob := rule.Bytes()
				copy(topic[common.HashLength-len(blob):], blob)
			case bool:
				if rule {
					topic[common.HashLength-1] = 1
				}
			case int8:
				blob := big.NewInt(int64(rule)).Bytes()
				copy(topic[common.HashLength-len(blob):], blob)
			case int16:
				blob := big.NewInt(int64(rule)).Bytes()
				copy(topic[common.HashLength-len(blob):], blob)
			case int32:
				blob := big.NewInt(int64(rule)).Bytes()
				copy(topic[common.HashLength-len(blob):], blob)
			case int64:
				blob := big.NewInt(rule).Bytes()
				copy(topic[common.HashLength-len(blob):], blob)
			case uint8:
				blob := new(big.Int).SetUint64(uint64(rule)).Bytes()
				copy(topic[common.HashLength-len(blob):], blob)
			case uint16:
				blob := new(big.Int).SetUint64(uint64(rule)).Bytes()
				copy(topic[common.HashLength-len(blob):], blob)
			case uint32:
				blob := new(big.Int).SetUint64(uint64(rule)).Bytes()
				copy(topic[common.HashLength-len(blob):], blob)
			case uint64:
				blob := new(big.Int).SetUint64(rule).Bytes()
				copy(topic[common.HashLength-len(blob):], blob)
			case string:
				hash := crypto.Keccak256Hash([]byte(rule))
				copy(topic[:], hash[:])
			case []byte:
				hash := crypto.Keccak256Hash(rule)
				copy(topic[:], hash[:])

			default:
				// Attempt to generate the topic from funky types
				val := reflect.ValueOf(rule)

				switch {
				case val.Kind() == reflect.Array && reflect.TypeOf(rule).Elem().Kind() == reflect.Uint8:
					reflect.Copy(reflect.ValueOf(topic[common.HashLength-val.Len():]), val)

				default:
					return nil, fmt.Errorf("unsupported indexed type: %T", rule)
				}
			}
			topics[i] = append(topics[i], topic)
		}
	}
	return topics, nil
}

// ensureContext is a helper method to ensure a context is not nil, even if the
// user specified it as such.
func ensureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.TODO()
	}
	return ctx
}
