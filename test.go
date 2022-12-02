package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

type Benchmark struct {
	Generator
	N int
}

func main() {
	cacheSize := []int{1e3, 10e3, 100e3, 1e6}
	multiplier := []int{10, 100, 1000}
	newCache := []NewCacheFunc{
		//NewHashicorpLRUV2,
		NewTinyLFUGenericResize,
		NewTinyLFUGenericNonResize,
	}
	newGen := []NewGeneratorFunc{
		NewScrambledZipfian,
		NewHotspot,
		NewUniform,
	}

	var results []*BenchmarkResult

	for _, newGen := range newGen {
		for _, cacheSize := range cacheSize {
			for _, multiplier := range multiplier {
				numKey := cacheSize * multiplier

				if len(results) > 0 {
					printResults(results)
					results = results[:0]
				}

				for _, newCache := range newCache {
					result := run(newGen, cacheSize, numKey, newCache)
					results = append(results, result)
				}
			}
		}
	}
}

func run(newGen NewGeneratorFunc, cacheSize, numKey int, newCache NewCacheFunc) *BenchmarkResult {
	gen := newGen(numKey)
	b := &Benchmark{
		Generator: gen,
		N:         1e6,
	}

	alloc1 := memAlloc()
	cache := newCache(cacheSize)
	defer cache.Close()

	start := time.Now()
	hits, misses := bench(b, cache)
	dur := time.Since(start)

	alloc2 := memAlloc()

	return &BenchmarkResult{
		GenName:   gen.Name(),
		CacheName: cache.Name(),
		CacheSize: cacheSize,
		NumKey:    numKey,

		Hits:     hits,
		Misses:   misses,
		Duration: dur,
		Bytes:    int64(alloc2) - int64(alloc1),
	}
}

func bench(b *Benchmark, cache Cache) (hits, misses int) {
	wg := sync.WaitGroup{}
	for i := 0; i < b.N; i++ {
		value := b.Next()
		//wg.Add(1)
		//go func() {
		//	defer wg.Done()
		if cache.Get(value) {
			hits++
		} else {
			misses++
			cache.Set(value)
		}
		//}()

	}

	tiny, ok := cache.(*TinyLFUGenericResize)

	if ok {
		fmt.Println(tiny.AllSizes())
		fmt.Println(tiny.AllKeys())
		fmt.Println(tiny.AllCaps())
	}

	wg.Wait()
	return hits, misses
}

func memAlloc() uint64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

type TinyLFUGenericResize struct {
	mu sync.Mutex
	v  *T[string]
}

type TinyLFUGenericNonResize struct {
	TinyLFUGenericResize
}

func NewTinyLFUGenericResize(size int) Cache {
	return &TinyLFUGenericResize{
		v: New[string](size, size*10, true),
	}
}
func NewTinyLFUGenericNonResize(size int) Cache {
	return &TinyLFUGenericNonResize{
		TinyLFUGenericResize{
			v: New[string](size, size*10, false),
		},
	}
}

func (c *TinyLFUGenericNonResize) Name() string {
	return "tinylfu-non"
}
func (c *TinyLFUGenericResize) Name() string {
	return "tinylfu-yes"
}

func (c *TinyLFUGenericResize) AllKeys() int {
	return c.v.AllKeys()
}

func (c *TinyLFUGenericResize) AllSizes() int {
	return c.v.AllSizes()
}

func (c *TinyLFUGenericResize) AllCaps() int {
	return c.v.AllCaps()
}

func (c *TinyLFUGenericResize) Set(key string) {
	c.mu.Lock()
	c.v.Add(key, key)
	c.mu.Unlock()
}

func (c *TinyLFUGenericResize) Get(key string) bool {
	c.mu.Lock()
	_, ok := c.v.Get(key)
	c.mu.Unlock()
	return ok
}

func (c *TinyLFUGenericResize) Close() {}
