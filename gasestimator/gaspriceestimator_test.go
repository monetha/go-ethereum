package gasestimator

import (
	"context"
	"math/big"
	"testing"
	"time"
)

func TestClose(t *testing.T) {
	e := newGasPriceEstimator(big.NewInt(1), newChanGasPrice(), 1*time.Microsecond)
	defer e.Close()

	e.Close()
}

func TestGasPriceEstimator_SuggestGasPrice(t *testing.T) {
	gasPricer := newChanGasPrice()
	e := newGasPriceEstimator(big.NewInt(1), gasPricer, 1*time.Microsecond)
	defer e.Close()

	updatePrice := big.NewInt(2)
	gasPricer.priceCh <- updatePrice
	// second update needed to make sure background goroutine updated internal state
	gasPricer.priceCh <- updatePrice

	price := e.SuggestGasPrice()
	if price.Cmp(updatePrice) != 0 {
		t.Fatalf("expected benchPrice %v, but got %v", updatePrice, price)
	}

	updatePrice = big.NewInt(3)
	gasPricer.priceCh <- updatePrice
	// second update needed to make sure background goroutine updated internal state
	gasPricer.priceCh <- updatePrice

	price = e.SuggestGasPrice()
	if price.Cmp(updatePrice) != 0 {
		t.Fatalf("expected benchPrice %v, but got %v", updatePrice, price)
	}
}

var benchPrice *big.Int

func BenchmarkGasPriceEstimator_SuggestGasPrice(b *testing.B) {
	e := newGasPriceEstimator(big.NewInt(1), newChanGasPrice(), 1*time.Microsecond)
	defer e.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchPrice = e.SuggestGasPrice()
	}
	b.StopTimer()
}

func newChanGasPrice() chanGasPrice {
	return chanGasPrice{make(chan *big.Int)}
}

type chanGasPrice struct{ priceCh chan *big.Int }

func (p chanGasPrice) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case price := <-p.priceCh:
		return price, nil
	}
}
