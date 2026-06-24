package domain

import (
	"math"
	"math/rand"
	"time"
)

// CalculateBackoff computes the delay before the next retry attempt
// using exponential backoff with jitter.
//
// Formula:
//
//	delay = min(baseBackoff * 2^(attempt-1), maxBackoff)
//	jitter = delay * jitterFactor * random(-1, 1)
//	result = delay + jitter
//
// The jitter prevents retry storms where many failed jobs retry
// at the same instant, which could overwhelm downstream services.
func CalculateBackoff(attempt int, policy RetryPolicy) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}

	// Exponential: baseBackoff * 2^(attempt-1)
	exp := math.Pow(2, float64(attempt-1))
	delay := time.Duration(float64(policy.BaseBackoff) * exp)

	// Cap at max backoff
	if delay > policy.MaxBackoff {
		delay = policy.MaxBackoff
	}

	// Apply jitter: +/- jitterFactor percent
	if policy.JitterFactor > 0 {
		jitterRange := float64(delay) * policy.JitterFactor
		// Random value in [-jitterRange, +jitterRange]
		jitter := time.Duration((rand.Float64()*2 - 1) * jitterRange)
		delay += jitter
	}

	// Never return a negative delay
	if delay < 0 {
		delay = 0
	}

	return delay
}

// NextRetryAt calculates the absolute time for the next retry attempt.
func NextRetryAt(now time.Time, attempt int, policy RetryPolicy) time.Time {
	return now.Add(CalculateBackoff(attempt, policy))
}
