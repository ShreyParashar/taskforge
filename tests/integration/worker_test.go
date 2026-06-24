package integration

// Worker integration tests will verify:
//   - Job leasing with FOR UPDATE SKIP LOCKED
//   - Heartbeat extension
//   - Stale lease rejection
//   - Graceful shutdown behavior