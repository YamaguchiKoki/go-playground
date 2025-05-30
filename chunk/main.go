package main

import (
	"fmt"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/samber/lo"
)

type Indices struct {
	Start, End int
}

// チャンク境界を計算する関数
func CalculateChunkBoundaries(totalLength, chunkingSize int) []Indices {
	if totalLength <= 0 {
		return []Indices{}
	}
	if chunkingSize <= 0 {
		return []Indices{
			{0, totalLength},
		}
	}

	if totalLength < chunkingSize {
		return []Indices{
			{0, totalLength},
		}
	}

	var result = []Indices{}
	for i := 0; i < totalLength; i += chunkingSize {
		if i+chunkingSize <= totalLength {
			result = append(result, Indices{i, i + chunkingSize})
		} else {
			result = append(result, Indices{i, totalLength})
		}
	}
	return result
}

// 直接アロケーションを行うチャンク関数
func AllocationBasedChunk[S ~[]T, T any](slice S, size int) [][]T {
	if len(slice) == 0 {
		return [][]T{}
	}

	var chunks [][]T
	for i := 0; i < len(slice); i += size {
		end := i + size
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

// チャンク境界を使ってスライスを分割する関数
func ChunkWithIndices[T any](slice []T, indices []Indices) [][]T {
	result := make([][]T, len(indices))
	for i, idx := range indices {
		result[i] = slice[idx.Start:idx.End]
	}
	return result
}

// ベンチマーク結果を保存する構造体
type BenchResult struct {
	Name           string
	Duration       time.Duration
	AllocatedBytes uint64
	Allocations    uint64
}

func runBenchmark(name string, iterations int, f func()) BenchResult {
	runtime.GC()

	// メモリ統計の初期値を取得
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// 時間計測開始
	start := time.Now()

	// 指定された回数だけ関数を実行
	for i := 0; i < iterations; i++ {
		f()
	}

	// 時間計測終了
	duration := time.Since(start)

	// メモリ統計の最終値を取得
	runtime.ReadMemStats(&m2)

	return BenchResult{
		Name:           name,
		Duration:       duration / time.Duration(iterations), // 1回あたりの平均時間
		AllocatedBytes: (m2.TotalAlloc - m1.TotalAlloc) / uint64(iterations), // 1回あたりの平均割り当てバイト数
		Allocations:    (m2.Mallocs - m1.Mallocs) / uint64(iterations), // 1回あたりの平均アロケーション回数
	}
}

func main() {
	fmt.Println("Goのチャンク関数ベンチマーク比較")
	fmt.Println("================================")

	// テストデータサイズとチャンクサイズ
	dataSizes := []int{100, 1000, 10000, 100000}
	chunkSizes := []int{10, 100, 1000}
	iterations := 1000 // 各ベンチマークの反復回数

	// 結果表示のヘッダー
	fmt.Printf("%-22s %-12s %-12s %-18s %-18s %-18s\n",
		"関数", "データサイズ", "チャンクサイズ", "実行時間/op", "メモリ(bytes)/op", "アロケ数/op")
	fmt.Println(strings.Repeat("-", 100))

	// 各サイズの組み合わせでベンチマークを実行
	for _, dataSize := range dataSizes {
		// テストデータを準備
		data := make([]int, dataSize)
		for i := range data {
			data[i] = i
		}

		for _, chunkSize := range chunkSizes {
			// チャンクサイズがデータサイズを超える場合はスキップ
			if chunkSize > dataSize {
				continue
			}

			// 1. slices.Chunk
			slicesResult := runBenchmark("slices.Chunk", iterations, func() {
				_ = slices.Chunk(data, chunkSize)
			})

			// 2. lo.Chunk
			loResult := runBenchmark("lo.Chunk", iterations, func() {
				_ = lo.Chunk(data, chunkSize)
			})

			// 3. CalculateChunkBoundaries + ChunkWithIndices
			indexResult := runBenchmark("インデックス計算型", iterations, func() {
				indices := CalculateChunkBoundaries(len(data), chunkSize)
				_ = ChunkWithIndices(data, indices)
			})

			// 4. AllocationBasedChunk
			allocResult := runBenchmark("直接アロケーション型", iterations, func() {
				_ = AllocationBasedChunk(data, chunkSize)
			})

			// 結果を出力
			for _, result := range []BenchResult{slicesResult, loResult, indexResult, allocResult} {
				fmt.Printf("%-22s %-12d %-12d %-18s %-18d %-18d\n",
					result.Name,
					dataSize,
					chunkSize,
					fmt.Sprintf("%v", result.Duration),
					result.AllocatedBytes,
					result.Allocations)
			}

			// 相対パフォーマンスを表示
			fmt.Println(strings.Repeat("-", 100))
			fmt.Printf("相対パフォーマンス（slices.Chunk = 1.0）:\n")
			fmt.Printf("  lo.Chunk          : 時間=%.2fx, メモリ=%.2fx, アロケ=%.2fx\n",
				float64(loResult.Duration)/float64(slicesResult.Duration),
				float64(loResult.AllocatedBytes)/float64(max(slicesResult.AllocatedBytes, 1)),
				float64(loResult.Allocations)/float64(max(slicesResult.Allocations, 1)))

			fmt.Printf("  インデックス計算型  : 時間=%.2fx, メモリ=%.2fx, アロケ=%.2fx\n",
				float64(indexResult.Duration)/float64(slicesResult.Duration),
				float64(indexResult.AllocatedBytes)/float64(max(slicesResult.AllocatedBytes, 1)),
				float64(indexResult.Allocations)/float64(max(slicesResult.Allocations, 1)))

			fmt.Printf("  直接アロケーション型: 時間=%.2fx, メモリ=%.2fx, アロケ=%.2fx\n",
				float64(allocResult.Duration)/float64(slicesResult.Duration),
				float64(allocResult.AllocatedBytes)/float64(max(slicesResult.AllocatedBytes, 1)),
				float64(allocResult.Allocations)/float64(max(slicesResult.Allocations, 1)))

			fmt.Println(strings.Repeat("-", 100))
			fmt.Println()
		}
	}
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
