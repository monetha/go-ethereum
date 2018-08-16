package ethereum

import (
	"context"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// SliceLogFilterer implements ethereum.SliceLogFilterer for log event slice.
type SliceLogFilterer []*types.Log

// FilterLogs implements ethereum.SliceLogFilterer.
func (logs SliceLogFilterer) FilterLogs(ctx context.Context, query ethereum.FilterQuery) (res []types.Log, err error) {
	topics := query.Topics

	res = make([]types.Log, 0, len(logs))
Logs:
	for _, log := range logs {
		if log == nil || log.Removed {
			continue
		}

		// If the to filtered topics is greater than the amount of topics in logs, skip.
		if len(topics) > len(log.Topics) {
			continue Logs
		}
		for i, sub := range topics {
			match := len(sub) == 0 // empty rule set == wildcard
			for _, topic := range sub {
				if log.Topics[i] == topic {
					match = true
					break
				}
			}
			if !match {
				continue Logs
			}
		}

		res = append(res, *log)
	}

	return
}

// SubscribeFilterLogs implements ethereum.SliceLogFilterer.
func (logs SliceLogFilterer) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	return event.NewSubscription(func(quit <-chan struct{}) error {
		logs, err := logs.FilterLogs(ctx, query)
		if err != nil {
			return err
		}

		for _, log := range logs {
			select {
			case ch <- log:
			case <-quit:
				return nil
			}
		}

		return nil
	}), nil
}
