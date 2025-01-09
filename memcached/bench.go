package main

import (
	"fmt"
	"log/slog"
	"math/rand/v2"
	"runtime"
	"sync"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

const (
	numKeys       = 1_000_000   // Total unique keys
	valueSize     = 100         // Size of each value in bytes
	randomKeyPool = 1_000_000   // Pool of random keys to generate high hit rate
	hitRate       = 0.95        // Desired hit rate
	getToSetRatio = 0.9         // Desired GET/SET ratio
	operations    = 100_000_000 // Total operations to perform
)

type operation struct {
	isGet bool
	key   string
	value string
}

func randomString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.IntN(len(letters))]
	}
	return string(b)
}

func main() {
	// Connect to Memcached
	mc := memcache.New("127.0.0.1:5001")
	if mc == nil {
		fmt.Println("Failed to create Memcached client")
		return
	}

	mc.MaxIdleConns = 1000
	mc.Timeout = 100 * time.Millisecond

	slog.Info("Connected to Memcached")

	// Prepopulate Memcached with keys to achieve the desired hit rate
	keyPool := make([]string, randomKeyPool)
	var wg sync.WaitGroup
	numPrepopulateWorkers := runtime.NumCPU() * 16
	keysPerWorker := randomKeyPool / numPrepopulateWorkers

	for w := 0; w < numPrepopulateWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			start := workerID * keysPerWorker
			end := start + keysPerWorker
			if workerID == numPrepopulateWorkers-1 {
				end = randomKeyPool
			}
			for i := start; i < end; i++ {
				key := fmt.Sprintf("key-%d", i)
				value := randomString(valueSize)

				// Only set keys with a probability of hitRate
				if rand.Float64() < hitRate {
					if err := mc.Set(&memcache.Item{Key: key, Value: []byte(value)}); err != nil {
						fmt.Printf("Failed to set key %s: %v\n", key, err)
					}
				}
				keyPool[i] = key
			}
		}(w)
	}

	wg.Wait()

	slog.Info("Generated key pool")

	numWorkers := runtime.NumCPU() * 8
	operationsCh := make(chan operation, operations)
	var getOps, setOps int64

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for op := range operationsCh {
				if op.isGet {
					// Perform GET operation
					_, err := mc.Get(op.key)
					if err == nil || err == memcache.ErrCacheMiss {
						getOps++
					} else {
						fmt.Printf("Failed to get key %s: %v\n", op.key, err)
					}
				} else {
					// Perform SET operation
					if err := mc.Set(&memcache.Item{Key: op.key, Value: []byte(op.value)}); err == nil {
						setOps++
					}
				}
			}
		}()
	}

	startTime := time.Now()

	// Generate operations
	for i := 0; i < operations; i++ {
		isGet := rand.Float64() < getToSetRatio
		if isGet {
			key := keyPool[rand.IntN(randomKeyPool)]
			operationsCh <- operation{isGet: true, key: key}
		} else {
			key := fmt.Sprintf("key-%d", rand.IntN(numKeys))
			value := randomString(valueSize)
			operationsCh <- operation{isGet: false, key: key, value: value}
		}
	}

	close(operationsCh)
	wg.Wait()
	duration := time.Since(startTime)
	fmt.Printf("Completed %d operations in %v\n", operations, duration)
	fmt.Printf("GET operations: %d, SET operations: %d\n", getOps, setOps)
	fmt.Printf("GET/SET ratio: %.2f\n", float64(getOps)/float64(setOps))
}
