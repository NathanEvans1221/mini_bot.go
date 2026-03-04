package agent

import (
	"log/slog"
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	var history []time.Time
	var ok bool
	if history, ok = rl.requests[key]; !ok {
		history = []time.Time{}
	}

	valid := history[:0]
	for _, t := range history {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.limit {
		slog.Warn("Rate limit exceeded", "key", key, "count", len(valid), "limit", rl.limit)
		return false
	}

	valid = append(valid, now)
	rl.requests[key] = valid
	return true
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		windowStart := now.Add(-rl.window)
		for key, history := range rl.requests {
			valid := history[:0]
			for _, t := range history {
				if t.After(windowStart) {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = valid
			}
		}
		rl.mu.Unlock()
	}
}

type UsageTracker struct {
	mu           sync.Mutex
	toolCalls    map[string]int
	totalCalls   int
	sessionCalls map[string]int
}

func NewUsageTracker() *UsageTracker {
	return &UsageTracker{
		toolCalls:    make(map[string]int),
		sessionCalls: make(map[string]int),
	}
}

func (ut *UsageTracker) RecordToolCall(session, toolName string) {
	ut.mu.Lock()
	defer ut.mu.Unlock()

	ut.totalCalls++
	ut.toolCalls[toolName]++
	ut.sessionCalls[session]++
}

func (ut *UsageTracker) GetStats() (total int, toolStats map[string]int, sessionStats map[string]int) {
	ut.mu.Lock()
	defer ut.mu.Unlock()

	toolStats = make(map[string]int)
	sessionStats = make(map[string]int)

	for k, v := range ut.toolCalls {
		toolStats[k] = v
	}
	for k, v := range ut.sessionCalls {
		sessionStats[k] = v
	}

	return ut.totalCalls, toolStats, sessionStats
}
