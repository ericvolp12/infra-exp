package main

import (
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	numKeys       = 10_000_000  // Total unique keys
	valueSize     = 100         // Size of each value in bytes
	randomKeyPool = 5_000_000   // Pool of random keys to generate high hit rate
	hitRate       = 0.95        // Desired hit rate
	getToSetRatio = 0.98        // Desired GET/SET ratio
	operations    = 100_000_000 // Total operations to perform
)

type operation struct {
	isGet bool
	key   string
	value string
}

var opDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "bench_memcached_op_duration_seconds",
	Help:    "Duration of Memcached operations",
	Buckets: prometheus.ExponentialBuckets(0.0001, 2, 20),
}, []string{"operation", "result"})

var getHits = opDuration.WithLabelValues("GET", "hit")
var getMisses = opDuration.WithLabelValues("GET", "miss")
var getError = opDuration.WithLabelValues("GET", "error")
var setOk = opDuration.WithLabelValues("SET", "ok")
var setError = opDuration.WithLabelValues("SET", "error")

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
	mc := memcache.New("127.0.0.1:11211")
	if mc == nil {
		fmt.Println("Failed to create Memcached client")
		return
	}

	// Start a metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":8754", nil)
	}()

	mc.MaxIdleConns = 100
	mc.Timeout = 100 * time.Millisecond

	slog.Info("Connected to Memcached")

	// Prepopulate Memcached with keys to achieve the desired hit rate
	keyPool := make([]string, randomKeyPool)
	var wg sync.WaitGroup
	numPrepopulateWorkers := 32
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
					s := time.Now()
					if err := mc.Set(&memcache.Item{Key: key, Value: []byte(value)}); err != nil {
						fmt.Printf("Failed to set key %s: %v\n", key, err)
						setError.Observe(time.Since(s).Seconds())
					} else {
						setOk.Observe(time.Since(s).Seconds())
					}
				}
				keyPool[i] = key
			}
		}(w)
	}

	wg.Wait()

	slog.Info("Generated key pool")

	numWorkers := 32
	operationsCh := make(chan operation, operations)
	var getOps, setOps int64

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for op := range operationsCh {
				s := time.Now()
				if op.isGet {
					// Perform GET operation
					_, err := mc.Get(op.key)
					if err != nil {
						if err == memcache.ErrCacheMiss {
							getMisses.Observe(time.Since(s).Seconds())
						} else {
							getError.Observe(time.Since(s).Seconds())
							fmt.Printf("Failed to get key %s: %v\n", op.key, err)
						}
					} else {
						getHits.Observe(time.Since(s).Seconds())
					}
					getOps++
				} else {
					// Perform SET operation
					if err := mc.Set(&memcache.Item{Key: op.key, Value: []byte(op.value)}); err != nil {
						fmt.Printf("Failed to set key %s: %v\n", op.key, err)
						setError.Observe(time.Since(s).Seconds())
					} else {
						setOk.Observe(time.Since(s).Seconds())
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
