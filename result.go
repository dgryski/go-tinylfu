package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"
)

type BenchmarkResult struct {
	GenName   string
	CacheName string
	CacheSize int
	NumKey    int

	Hits     int
	Misses   int
	Duration time.Duration
	Bytes    int64
}

func printResults(results []*BenchmarkResult) {
	res := results[0]
	fmt.Printf("%s cache=%d keys=%d\n\n", res.GenName, res.CacheSize, res.NumKey)

	var data [][]string

	for _, res := range results {
		hitRate := float64(res.Hits) / float64(res.Hits+res.Misses)
		ms := res.Duration.Milliseconds()

		data = append(data, []string{
			res.CacheName,
			strconv.FormatFloat(hitRate, 'f', 2, 64),
			strconv.FormatInt(ms, 10) + "ms",
			formatBytes(res.Bytes),
			strconv.Itoa(res.Hits),
			strconv.Itoa(res.Misses),
		})
	}

	renderTable(data)
	fmt.Printf("\n\n")
}

func renderTable(data [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"cache", "hit rate", "time", "memory", "hits", "misses"})
	table.SetColumnAlignment([]int{
		tablewriter.ALIGN_LEFT,
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_RIGHT,
	})
	table.SetBorder(false)
	table.AppendBulk(data)
	table.Render()
}

func formatBytes(n int64) string {
	if n < 0 {
		return "-"
	}
	return strconv.FormatFloat(float64(n)/1024/1024, 'f', 2, 64) + "MiB"
}
