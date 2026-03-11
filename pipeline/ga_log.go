package pipeline

import (
	"fmt"
	"sync"
	"time"
)

// GALogger provides thread-safe, rate-limited logging for GA events.
// It shares a mutex with the assembly progress bar to prevent terminal
// corruption from interleaved \r\033[K sequences.
// A nil *GALogger is safe to use — all methods are no-ops.
type GALogger struct {
	mu       *sync.Mutex
	lastLog  time.Time
	minDelay time.Duration
	start    time.Time
	verbose  bool
}

// NewGALogger creates a GALogger. If verbose is false, all logging is suppressed.
// mu must be the same mutex used by the progress bar in AssembleConcurrently.
func NewGALogger(mu *sync.Mutex, verbose bool) *GALogger {
	return &GALogger{
		mu:       mu,
		minDelay: 500 * time.Millisecond,
		start:    time.Now(),
		verbose:  verbose,
	}
}

// LogOptimum logs a new global best fitness discovery.
// Rate-limited to avoid flooding the terminal during rapid early generations.
func (l *GALogger) LogOptimum(islandN, fitness int) {
	if l == nil || !l.verbose {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if now.Sub(l.lastLog) < l.minDelay {
		return
	}
	l.lastLog = now
	elapsed := now.Sub(l.start).Truncate(time.Millisecond)
	fmt.Printf("\r\033[K  [GA] n=%d  new optimum: fitness=%d  (%v)\n", islandN, fitness, elapsed)
}

// LogStagnation logs when a worker enters panic-mutation mode.
func (l *GALogger) LogStagnation(islandN, stagnation int) {
	if l == nil || !l.verbose {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if now.Sub(l.lastLog) < l.minDelay {
		return
	}
	l.lastLog = now
	fmt.Printf("\r\033[K  [GA] n=%d  stagnation=%d, applying heavy mutations\n", islandN, stagnation)
}

// LogRemix logs when chunk remixing is triggered.
func (l *GALogger) LogRemix(badChunks, totalStrings int) {
	if l == nil || !l.verbose {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lastLog = time.Now()
	fmt.Printf("\r\033[K  [REMIX] %d chunks underperforming, remixing %d strings\n", badChunks, totalStrings)
}
