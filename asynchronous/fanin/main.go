package main

import (
	"fmt"
	"sync"
	"time"
)

func fanIn[T any](done <-chan struct{}, channels ...<-chan T) <-chan T {
	out := make(chan T)
	var wg sync.WaitGroup

	multiplex := func(c <-chan T) {
		defer wg.Done()
		for n := range c {
			select {
			case <-done:
				return
			case out <- n:
			}
		}
	}

	wg.Add(len(channels))
	for _, c := range channels {
		go multiplex(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func main() {
	done := make(chan struct{})
	defer close(done)

	// 3つの入力チャネルを作成
	ch1 := make(chan int)
	ch2 := make(chan int)
	ch3 := make(chan int)

	// fanInで統合
	merged := fanIn(done, ch1, ch2, ch3)

	// 結果を収集するgoroutine
	results := make([]int, 0)
	var resultMutex sync.Mutex

	go func() {
		for val := range merged {
			resultMutex.Lock()
			results = append(results, val)
			fmt.Printf("受信: %d\n", val)
			resultMutex.Unlock()
		}
	}()

	// 各チャネルにデータを送信
	go func() {
		defer close(ch1)
		for i := 1; i <= 3; i++ {
			ch1 <- i * 10
			time.Sleep(100 * time.Millisecond)
		}
	}()

	go func() {
		defer close(ch2)
		for i := 1; i <= 3; i++ {
			ch2 <- i * 100
			time.Sleep(150 * time.Millisecond)
		}
	}()

	go func() {
		defer close(ch3)
		for i := 1; i <= 2; i++ {
			ch3 <- i * 1000
			time.Sleep(200 * time.Millisecond)
		}
	}()

	time.Sleep(1 * time.Second)

	resultMutex.Lock()
	fmt.Printf("✅ 受信した値の数: %d\n", len(results))
	fmt.Printf("   期待値: 8個 (3+3+2)\n\n")
	resultMutex.Unlock()
}
