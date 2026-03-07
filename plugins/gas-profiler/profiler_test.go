package gasprofiler

import (
	"context"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	p := New()
	require.NotNil(t, p)
	assert.NotNil(t, p.methods)
	assert.Empty(t, p.methods)
}

func TestRecord_SingleEntry(t *testing.T) {
	p := New()
	p.Record("TokenA", "transfer", 21000)

	key := "TokenA.transfer"
	mp, ok := p.methods[key]
	require.True(t, ok)
	assert.Equal(t, "TokenA", mp.Contract)
	assert.Equal(t, "transfer", mp.Method)
	assert.Equal(t, []uint64{21000}, mp.Samples)
}

func TestRecord_MultipleSamples(t *testing.T) {
	p := New()
	p.Record("TokenA", "transfer", 21000)
	p.Record("TokenA", "transfer", 30000)
	p.Record("TokenA", "transfer", 25000)

	mp := p.methods["TokenA.transfer"]
	require.NotNil(t, mp)
	assert.Len(t, mp.Samples, 3)
	assert.Equal(t, []uint64{21000, 30000, 25000}, mp.Samples)
}

func TestRecord_MultipleMethods(t *testing.T) {
	p := New()
	p.Record("TokenA", "transfer", 21000)
	p.Record("TokenA", "approve", 45000)
	p.Record("TokenB", "transfer", 65000)

	assert.Len(t, p.methods, 3)
	assert.Contains(t, p.methods, "TokenA.transfer")
	assert.Contains(t, p.methods, "TokenA.approve")
	assert.Contains(t, p.methods, "TokenB.transfer")
}

func TestStats_Empty(t *testing.T) {
	p := New()
	stats := p.Stats()
	assert.Empty(t, stats)
}

func TestStats_SingleSample(t *testing.T) {
	p := New()
	p.Record("Token", "transfer", 50000)

	stats := p.Stats()
	require.Len(t, stats, 1)
	s := stats[0]
	assert.Equal(t, "Token", s.Contract)
	assert.Equal(t, "transfer", s.Method)
	assert.Equal(t, uint64(50000), s.MinGas)
	assert.Equal(t, uint64(50000), s.AvgGas)
	assert.Equal(t, uint64(50000), s.MaxGas)
	assert.Equal(t, 1, s.Calls)
}

func TestStats_MultipleSamples(t *testing.T) {
	p := New()
	p.Record("Token", "mint", 100000)
	p.Record("Token", "mint", 200000)
	p.Record("Token", "mint", 150000)

	stats := p.Stats()
	require.Len(t, stats, 1)
	s := stats[0]
	assert.Equal(t, uint64(100000), s.MinGas)
	assert.Equal(t, uint64(150000), s.AvgGas) // (100000+200000+150000)/3 = 150000
	assert.Equal(t, uint64(200000), s.MaxGas)
	assert.Equal(t, 3, s.Calls)
}

func TestStats_SortedByAvgGasDescending(t *testing.T) {
	p := New()
	p.Record("C", "low", 10000)
	p.Record("B", "mid", 50000)
	p.Record("A", "high", 100000)

	stats := p.Stats()
	require.Len(t, stats, 3)
	assert.Equal(t, uint64(100000), stats[0].AvgGas)
	assert.Equal(t, uint64(50000), stats[1].AvgGas)
	assert.Equal(t, uint64(10000), stats[2].AvgGas)
}

func TestStats_AvgGasIntegerDivision(t *testing.T) {
	p := New()
	p.Record("X", "foo", 10)
	p.Record("X", "foo", 20)
	p.Record("X", "foo", 33)

	stats := p.Stats()
	require.Len(t, stats, 1)
	assert.Equal(t, uint64(21), stats[0].AvgGas)
}

func TestSuggest_NoneAboveThreshold(t *testing.T) {
	p := New()
	p.Record("Token", "transfer", 21000)

	suggestions := p.Suggest(100000)
	assert.Empty(t, suggestions)
}

func TestSuggest_SomeAboveThreshold(t *testing.T) {
	p := New()
	p.Record("Token", "transfer", 21000)
	p.Record("DEX", "swap", 250000)

	suggestions := p.Suggest(100000)
	require.Len(t, suggestions, 1)
	assert.Contains(t, suggestions[0], "DEX.swap")
	assert.Contains(t, suggestions[0], "consider optimizing storage patterns")
}

func TestSuggest_AllAboveThreshold(t *testing.T) {
	p := New()
	p.Record("A", "foo", 200000)
	p.Record("B", "bar", 300000)

	suggestions := p.Suggest(100000)
	assert.Len(t, suggestions, 2)
}

func TestSuggest_ExactlyAtThreshold(t *testing.T) {
	p := New()
	p.Record("Token", "transfer", 100000)

	suggestions := p.Suggest(100000)
	assert.Empty(t, suggestions)
}

func TestReset(t *testing.T) {
	p := New()
	p.Record("A", "foo", 100)
	p.Record("B", "bar", 200)
	require.NotEmpty(t, p.Stats())

	p.Reset()
	assert.Empty(t, p.Stats())
	assert.Empty(t, p.methods)
}

func TestReset_CanRecordAfterReset(t *testing.T) {
	p := New()
	p.Record("A", "foo", 100)
	p.Reset()
	p.Record("B", "bar", 200)

	stats := p.Stats()
	require.Len(t, stats, 1)
	assert.Equal(t, "B", stats[0].Contract)
}

func TestOnTransaction(t *testing.T) {
	p := New()
	ctx := context.Background()
	p.OnTransaction(ctx, "Token", "transfer", 21000)

	stats := p.Stats()
	require.Len(t, stats, 1)
	assert.Equal(t, "Token", stats[0].Contract)
	assert.Equal(t, "transfer", stats[0].Method)
	assert.Equal(t, uint64(21000), stats[0].AvgGas)
}

func TestConcurrentRecord(t *testing.T) {
	p := New()
	const goroutines = 50
	const recordsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < recordsPerGoroutine; i++ {
				p.Record("Contract", "method", uint64(i+id*1000))
			}
		}(g)
	}
	wg.Wait()

	stats := p.Stats()
	require.Len(t, stats, 1)
	assert.Equal(t, goroutines*recordsPerGoroutine, stats[0].Calls)
}

func TestConcurrentRecordAndStats(t *testing.T) {
	p := New()
	const goroutines = 20

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				p.Record("C", "m", uint64(i))
			}
		}(g)
	}

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				_ = p.Stats()
			}
		}()
	}

	wg.Wait()
	stats := p.Stats()
	require.Len(t, stats, 1)
	assert.Equal(t, goroutines*100, stats[0].Calls)
}

func TestConcurrentRecordAndReset(t *testing.T) {
	p := New()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			p.Record("C", "m", uint64(i))
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			p.Reset()
		}
	}()

	wg.Wait()
}

func TestRecord_ZeroGas(t *testing.T) {
	p := New()
	p.Record("C", "m", 0)

	stats := p.Stats()
	require.Len(t, stats, 1)
	assert.Equal(t, uint64(0), stats[0].MinGas)
	assert.Equal(t, uint64(0), stats[0].AvgGas)
	assert.Equal(t, uint64(0), stats[0].MaxGas)
}

func TestRecord_EmptyContractAndMethod(t *testing.T) {
	p := New()
	p.Record("", "", 42)

	key := "."
	mp, ok := p.methods[key]
	require.True(t, ok)
	assert.Equal(t, uint64(42), mp.Samples[0])
}

func TestStats_Deterministic_Sort(t *testing.T) {
	p := New()
	p.Record("A", "x", 100)
	p.Record("B", "y", 100)

	stats := p.Stats()
	require.Len(t, stats, 2)
	assert.Equal(t, uint64(100), stats[0].AvgGas)
	assert.Equal(t, uint64(100), stats[1].AvgGas)
	assert.True(t, sort.SliceIsSorted(stats, func(i, j int) bool {
		return stats[i].AvgGas > stats[j].AvgGas
	}))
}
