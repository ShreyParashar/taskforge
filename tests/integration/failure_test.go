package integration

// Failure scenario tests will verify:
//   - Worker crash recovery via reaper
//   - API timeout retry behavior
//   - Stale lease token rejection
//   - Dead-letter queue behavior
//   - Rate limiting under load
//   - Circuit breaker state transitions