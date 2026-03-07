package gasprofiler

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

type Profiler struct {
	mu      sync.Mutex
	methods map[string]*MethodProfile
}

type MethodProfile struct {
	Contract string   `json:"contract"`
	Method   string   `json:"method"`
	Samples  []uint64 `json:"samples"`
}

type Stats struct {
	Contract string `json:"contract"`
	Method   string `json:"method"`
	MinGas   uint64 `json:"min_gas"`
	AvgGas   uint64 `json:"avg_gas"`
	MaxGas   uint64 `json:"max_gas"`
	Calls    int    `json:"calls"`
}

func New() *Profiler {
	return &Profiler{methods: make(map[string]*MethodProfile)}
}

func (p *Profiler) Record(contract, method string, gasUsed uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := contract + "." + method
	mp, ok := p.methods[key]
	if !ok {
		mp = &MethodProfile{Contract: contract, Method: method}
		p.methods[key] = mp
	}
	mp.Samples = append(mp.Samples, gasUsed)
}

func (p *Profiler) Stats() []Stats {
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make([]Stats, 0, len(p.methods))
	for _, mp := range p.methods {
		if len(mp.Samples) == 0 {
			continue
		}

		var min, max, total uint64
		min = mp.Samples[0]
		for _, s := range mp.Samples {
			total += s
			if s < min {
				min = s
			}
			if s > max {
				max = s
			}
		}

		result = append(result, Stats{
			Contract: mp.Contract,
			Method:   mp.Method,
			MinGas:   min,
			AvgGas:   total / uint64(len(mp.Samples)),
			MaxGas:   max,
			Calls:    len(mp.Samples),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].AvgGas > result[j].AvgGas
	})

	return result
}

func (p *Profiler) Suggest(threshold uint64) []string {
	stats := p.Stats()
	var suggestions []string
	for _, s := range stats {
		if s.AvgGas > threshold {
			suggestions = append(suggestions, fmt.Sprintf(
				"%s.%s: avg %d gas (%d calls) — consider optimizing storage patterns",
				s.Contract, s.Method, s.AvgGas, s.Calls,
			))
		}
	}
	return suggestions
}

func (p *Profiler) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.methods = make(map[string]*MethodProfile)
}

func (p *Profiler) OnTransaction(ctx context.Context, contract, method string, gasUsed uint64) {
	p.Record(contract, method, gasUsed)
}
