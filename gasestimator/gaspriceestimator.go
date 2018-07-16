package gasestimator

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/ethclient"
)

// GasPriceEstimator is the gas price estimator, it returns cached gas price to allow a timely
// execution of a transaction.
type GasPriceEstimator struct {
	gasPrice       *big.Int
	gasPricer      ethereum.GasPricer
	updateInterval time.Duration
	rwMutex        sync.RWMutex
	wg             sync.WaitGroup
	closeOnce      sync.Once
	closed         chan struct{}
}

// NewGasPriceEstimator creates an instance of GasPriceEstimator
func NewGasPriceEstimator(rawRPCURL string) (*GasPriceEstimator, error) {
	cl, err := ethclient.Dial(rawRPCURL)
	if err != nil {
		return nil, fmt.Errorf("gasestimator: ethclient.Dial: %v", err)
	}

	gasPrice, err := cl.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("gasestimator: SuggestGasPrice: %v", err)
	}

	return newGasPriceEstimator(gasPrice, cl, 4*time.Second), nil
}

func newGasPriceEstimator(initGasPrice *big.Int, gasPricer ethereum.GasPricer, updateInterval time.Duration) *GasPriceEstimator {
	estimator := &GasPriceEstimator{
		gasPrice:       initGasPrice,
		gasPricer:      gasPricer,
		updateInterval: updateInterval,
		closed:         make(chan struct{}),
	}
	estimator.runAsync()

	return estimator
}

// Close implements io.Closer interface.
func (e *GasPriceEstimator) Close() (err error) {
	e.closeOnce.Do(func() {
		close(e.closed)

		e.wg.Wait()
	})

	return
}

// SuggestGasPrice retrieves the currently suggested gas price to allow a timely
// execution of a transaction.
func (e *GasPriceEstimator) SuggestGasPrice() (gasPrice *big.Int) {
	e.rwMutex.RLock()
	gasPrice = new(big.Int).Set(e.gasPrice)
	e.rwMutex.RUnlock()
	return
}

func (e *GasPriceEstimator) runAsync() {
	ctx, cancel := context.WithCancel(context.Background())
	e.cancelOnClose(cancel)

	e.wg.Add(1)
	go func() {
		defer e.wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(e.updateInterval):
			}

			newGasPrice, err := e.gasPricer.SuggestGasPrice(ctx)
			if err != nil {
				log.Printf("gasestimator: SuggestGasPrice: %v", err)
				continue
			}
			curGasPrice := e.SuggestGasPrice()

			if curGasPrice.Cmp(newGasPrice) != 0 {
				e.rwMutex.Lock()
				e.gasPrice = newGasPrice
				e.rwMutex.Unlock()
			}
		}
	}()
}

func (e *GasPriceEstimator) cancelOnClose(cancel context.CancelFunc) {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		defer cancel()

		<-e.closed
	}()
}
