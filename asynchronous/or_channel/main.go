package main

import (
	"fmt"
	"time"
)


// case数はコンパイル時に決定されないといけないが、可変個のチャネルをまとめたい→分割統治
func or (channels ...<-chan any) <-chan any {
	// base case
	switch len(channels) {
	case 0:
		return nil
	case 1:
		return channels[0]
	}

	orDone := make(chan any)
	go func() {
		defer close(orDone)

		switch len(channels) {
		// 再帰呼び出し時のインデックス境界の安全性を担保する
		case 2:
			select {
			case <- channels[0]:
			case <- channels[1]:
			}
		default:
			select {
			case <- channels[0]:
			case <- channels[1]:
			case <- channels[2]:
			case <-or(append(channels[3:], orDone)...): // 隣接する再帰呼び出しをorDoneチャネルで接続する
			}
		}
	}()
	return orDone
}

func main() {
	sig := func(after time.Duration) <-chan any {
		fmt.Println(after)
		c := make(chan any)
		go func() {
			defer close(c)
			time.Sleep(after)
		}()
		return c
	}

	start := time.Now()
	<-or(
		sig(1 * time.Hour),
		sig(1 * time.Minute),
		sig(1 * time.Second),
		sig(1 * time.Minute),
	)

	// 1sを期待
	fmt.Printf("done after %v", time.Since(start))
}

