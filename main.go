package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	flagThreads = flag.Int("threads", 1, "number of concurrent processors")
	flagSize    = flag.Int("size", 10000, "document length")
	flagCount   = flag.Int("count", 0, "number of transactions to run (0=unlimited)")
	flagHosts   = flag.String("hosts", "localhost:9080", "dgraph alpha host(s)")
)

func main() {
	flag.Parse()

	ctx := rootContext()
	dg, err := newDgraph(strings.Split(*flagHosts, ","))
	if err != nil {
		log.Fatal(err)
	}

	err = dg.setup(ctx)
	if err != nil {
		log.Fatal(err)
	}

	var (
		batchSize = *flagCount / *flagThreads
		batchLeft = *flagCount % *flagThreads
		ch        = make(chan *timingSet)
		wg        sync.WaitGroup
	)
	go trackProgress(ctx, ch)
	for i := 0; i < *flagThreads; i++ {
		size := batchSize
		if i == 0 {
			size += batchLeft
		}
		if size == 0 && *flagCount != 0 {
			continue
		}
		wg.Add(1)
		go func() {
			runBatch(ctx, dg, size, ch)
			wg.Done()
		}()
	}
	wg.Wait()
}

func runBatch(ctx context.Context, dg *dgraph, n int, ch chan *timingSet) {
	for i := 0; i < n || n == 0; i++ {
		t, err := dg.run(ctx, *flagSize)
		if err != nil {
			log.Fatal(err)
		}
		go func() { ch <- t }()
	}
}

func trackProgress(ctx context.Context, ch chan *timingSet) {
	var (
		count = 0
		total = time.Duration(0)
		start = time.Now()
		last  *timingSet
	)
	for {
		select {
		case <-ctx.Done():
			fmt.Println()
			return
		case last = <-ch:
			count++
			total += last.Total()
		}

	consume:
		for {
			select {
			case <-ctx.Done():
				fmt.Println()
				return
			case last = <-ch:
				count++
				total += last.Total()
			default:
				break consume
			}
		}

		avg := (total / time.Duration(count)).Round(time.Microsecond)
		rate := int(float64(count) / time.Since(start).Seconds())
		fmt.Printf("\033[2K\rcount: %5d avg: %-8v rate: %-4d/s last: %v", count, avg, rate, last)
	}
}

func rootContext() context.Context {
	var (
		ch        = make(chan os.Signal, 1)
		ctx, done = context.WithCancel(context.Background())
	)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch
		_, _ = fmt.Fprintln(os.Stderr, "Received interrupt, shutting down...")
		done()

		<-ch
		_, _ = fmt.Fprintln(os.Stderr, "Received interrupt during shutdown, exiting...")
		os.Exit(1)
	}()
	return ctx
}
