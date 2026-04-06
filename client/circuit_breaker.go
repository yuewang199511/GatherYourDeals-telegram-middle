package client

import (
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen is returned when the circuit breaker is open and requests are rejected.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CBConfig holds circuit breaker tunables, typically loaded from environment variables.
type CBConfig struct {
	// FailureThreshold is the number of consecutive failures required to open the circuit.
	FailureThreshold int
	// SuccessThreshold is the number of consecutive successes in half-open required to close.
	SuccessThreshold int
	// OpenTimeout is how long to stay open before allowing a probe request.
	OpenTimeout time.Duration
}

type cbState int

const (
	cbStateClosed   cbState = iota // normal operation
	cbStateOpen                    // rejecting requests after too many failures
	cbStateHalfOpen                // probing whether the upstream has recovered
)

// CircuitBreaker protects outbound calls from a repeatedly failing upstream.
//
// Transitions:
//
//	Closed  --[failures >= threshold]--> Open
//	Open    --[openTimeout elapsed]----> Half-Open
//	Half-Open --[success >= threshold]-> Closed
//	Half-Open --[any failure]----------> Open
type CircuitBreaker struct {
	mu               sync.Mutex
	state            cbState
	failureCount     int
	successCount     int
	failureThreshold int
	successThreshold int
	openTimeout      time.Duration
	openedAt         time.Time
}

func newCircuitBreaker(cfg CBConfig) *CircuitBreaker {
	return &CircuitBreaker{
		state:            cbStateClosed,
		failureThreshold: cfg.FailureThreshold,
		successThreshold: cfg.SuccessThreshold,
		openTimeout:      cfg.OpenTimeout,
	}
}

// allow returns true if the call should proceed.
// When Open, it transitions to Half-Open once the timeout has elapsed.
func (cb *CircuitBreaker) allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case cbStateClosed:
		return true
	case cbStateOpen:
		if time.Since(cb.openedAt) >= cb.openTimeout {
			cb.state = cbStateHalfOpen
			cb.successCount = 0
			return true
		}
		return false
	case cbStateHalfOpen:
		return true
	}
	return false
}

// recordSuccess resets the failure counter and, in Half-Open, may close the circuit.
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0
	if cb.state == cbStateHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.state = cbStateClosed
		}
	}
}

// recordFailure increments the failure counter and may open the circuit.
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == cbStateHalfOpen {
		cb.state = cbStateOpen
		cb.openedAt = time.Now()
		return
	}

	cb.failureCount++
	if cb.failureCount >= cb.failureThreshold {
		cb.state = cbStateOpen
		cb.openedAt = time.Now()
		cb.failureCount = 0
	}
}
