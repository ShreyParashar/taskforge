package worker

// Heartbeat logic will be implemented in Phase 5.
// Its responsibility is to periodically extend the locked_until
// timestamp on jobs that are actively running in the Executor,
// preventing the Reaper from assuming the worker crashed.
