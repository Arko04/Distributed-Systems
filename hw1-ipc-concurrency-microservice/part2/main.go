package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

type Workload int

const (
	CPU Workload = iota
	Mixed
)

func cpuWork() {
	sum := 0.0
	for i := 0; i < 10_000_000; i++ {
		sum += float64(i) * 1.0001
	}
	_ = sum
}

func mixedWork(mu *sync.Mutex, counter *int) {
	sum := 0
	for i := 0; i < 100_000; i++ {
		sum += i
	}
	mu.Lock()
	*counter++
	mu.Unlock()
	_ = sum
}

// runOnce runs a single benchmark for given settings and returns duration + throughput.
func runOnce(goroutines int, w Workload, procs int) (time.Duration, float64) {
	// Limit the number of OS threads that can execute Go code simultaneously.
	runtime.GOMAXPROCS(procs)

	var wg sync.WaitGroup
	start := time.Now()

	if w == CPU {
		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cpuWork()
			}()
		}
	} else {
		var mu sync.Mutex
		counter := 0
		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				mixedWork(&mu, &counter)
			}()
		}
	}

	wg.Wait()
	total := time.Since(start)
	throughput := float64(goroutines) / total.Seconds()
	return total, throughput
}

func benchmarkAvg(goroutines int, w Workload, procs, runs int) (time.Duration, float64) {
	var totalDur time.Duration
	var totalThroughput float64

	for i := 0; i < runs; i++ {
		d, tp := runOnce(goroutines, w, procs)
		totalDur += d
		totalThroughput += tp
	}

	avgDur := totalDur / time.Duration(runs)
	avgTp := totalThroughput / float64(runs)

	return avgDur, avgTp
}

func main() {
	const runsPerConfig = 5

	counts := []int{1, 2, 4, 8, 16, 32, 64}
	procsList := []int{1, 2, runtime.NumCPU()}
	workloads := []struct {
		name string
		typ  Workload
	}{
		{"CPU-bound", CPU},
		{"Mixed", Mixed},
	}

	fmt.Println("Goroutines,Workload,GOMAXPROCS,Runs,TotalTimeNs,TotalTimeMs,ThroughputReqPerSec")

	for _, g := range counts {
		for _, w := range workloads {
			for _, p := range procsList {
				avgDur, avgTp := benchmarkAvg(g, w.typ, p, runsPerConfig)
				ms := float64(avgDur.Milliseconds())
				fmt.Printf("%d,%s,%d,%d,%d,%.2f,%.2f\n",
					g, w.name, p, runsPerConfig,
					avgDur.Nanoseconds(), ms, avgTp)
			}
		}
	}
}
