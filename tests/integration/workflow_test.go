package integration

// Workflow integration tests will verify:
//   - DAG validation (cycles, missing dependencies)
//   - Step execution order based on dependencies
//   - Workflow completion when all steps succeed
//   - Workflow failure when a required step dead-letters
//   - Workflow cancellation propagation