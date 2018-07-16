package blocksource

import (
	"context"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/monetha/go-ethereum"
	"github.com/monetha/go-ethereum/client"
)

// Config contains parameters of BlockSource.
type Config struct {
	// StartBlock is the number of the block from which to start the delivery of blocks.
	// If number is nil, the latest known block that has specified number of confirmations
	// is used as start block.
	StartBlock *big.Int
	// Confirmations number indicates that the block must be delivered only when it has
	// the specified number of confirmations (number of blocks mined since delivered block).
	Confirmations uint
}

// BlockSource holds a channel that delivers blocks from Ethereum channel.
type BlockSource struct {
	C         <-chan *ethereum.Block // The channel on which the blocks are delivered.
	client    *client.Client
	wg        sync.WaitGroup
	closeOnce sync.Once
	closed    chan struct{}
}

// New returns a new BlockSource containing a channel that will deliver the blocks from Ethereum network.
func New(rawurl string, cfg *Config) (*BlockSource, error) {
	cl, err := client.Dial(rawurl)
	if err != nil {
		return nil, err
	}

	ch := make(chan *ethereum.Block)
	bs := &BlockSource{
		C:      ch,
		client: cl,
		closed: make(chan struct{}),
	}

	if cfg == nil {
		cfg = &Config{}
	}
	bs.runAsync(cfg, ch)

	return bs, nil
}

// Blocks returns the channel on which the blocks are delivered.
func (bs *BlockSource) Blocks() <-chan *ethereum.Block {
	return bs.C
}

// Close implements io.Closer interface.
func (bs *BlockSource) Close() (err error) {
	bs.closeOnce.Do(func() {
		close(bs.closed)

		bs.wg.Wait()

		err = bs.client.Close()
	})

	return
}

func (bs *BlockSource) runAsync(cfg *Config, blocks chan *ethereum.Block) {
	ctx, cancel := context.WithCancel(context.Background())
	bs.cancelOnClose(cancel)

	bs.wg.Add(1)
	go func() {
		defer bs.wg.Done()
		defer close(blocks) // close blocks when closed

		one := big.NewInt(1)

		var recentBlkNumber *big.Int
		var currBlkNumber *big.Int
		if cfg.StartBlock != nil {
			currBlkNumber = new(big.Int).Set(cfg.StartBlock) // copy start block number
		}
		confirmations := big.NewInt(int64(cfg.Confirmations))

		delayBeforeIteration := false
		for {
			if delayBeforeIteration {
				select {
				case <-ctx.Done():
					return
				case <-time.After(4 * time.Second):
					delayBeforeIteration = false
				}
			}

			if needToGetMostRecentBlockNumber(currBlkNumber, recentBlkNumber, confirmations) {
				var err error
				recentBlkNumber, err = bs.client.BlockNumber(ctx)
				if err != nil {
					log.Printf("BlockNumber: %v", err)
					delayBeforeIteration = true
					continue
				}
				if needToGetMostRecentBlockNumber(currBlkNumber, recentBlkNumber, confirmations) {
					delayBeforeIteration = true
					continue
				}
			}

			if recentBlkNumber != nil && currBlkNumber == nil {
				currBlkNumber = new(big.Int).Sub(recentBlkNumber, confirmations)
			}

			b, err := bs.client.BlockByNumber(ctx, currBlkNumber)
			if err != nil {
				if err != ethereum.ErrNotFound { // when block isn't found it's ok, we just need to wait more
					log.Printf("BlockByNumber: %v", err)
				}
				delayBeforeIteration = true
				continue
			} else {
				// increment currBlkNumber
				currBlkNumber = new(big.Int).Add(b.Number, one)

				// deliver new block
				select {
				case <-ctx.Done():
					return
				case blocks <- b:
				}
			}
		}
	}()
}

func needToGetMostRecentBlockNumber(currentBlockNumber, recentBlockNumber, confirmations *big.Int) bool {
	// confirmations > 0
	return confirmations.Sign() == 1 &&
		(recentBlockNumber == nil ||
			// recentBlkNumber < confirmations
			recentBlockNumber.Cmp(confirmations) == -1 ||
			// currBlkNumber != nil && currBlkNumber + confirmations > recentBlkNumber
			currentBlockNumber != nil && new(big.Int).Add(currentBlockNumber, confirmations).Cmp(recentBlockNumber) == 1)
}

func (bs *BlockSource) cancelOnClose(cancel context.CancelFunc) {
	bs.wg.Add(1)
	go func() {
		defer bs.wg.Done()
		defer cancel()

		<-bs.closed
	}()
}
