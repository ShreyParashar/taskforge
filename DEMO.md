# TaskForge — Live Demo Proof ✅

> All output below is **real, live data** from a running TaskForge instance on `localhost`.

---

## 🟢 Step 1 — API Health Check

```
GET http://localhost:8080/healthz
→ 200 OK

{"status":"ok"}
```

---

## 🟢 Step 2 — Create Namespace

```
POST http://localhost:8080/v1/namespaces
Body: {"name":"production"}
→ 201 Created

{
  "id":   "bd59caad-7080-445e-8128-d4a3c7064298",
  "name": "production",
  "created_at": "2026-06-24T12:54:29+05:30"
}
```

---

## 🟢 Step 3 — Create Queue

```
POST http://localhost:8080/v1/namespaces/production/queues
Body: {"name":"jobs","concurrency_limit":10}
→ 201 Created

{
  "id":                "f71bbf36-6ee2-42c1-90d9-e257dba72310",
  "namespace_id":      "bd59caad-7080-445e-8128-d4a3c7064298",
  "name":              "jobs",
  "concurrency_limit": 10
}
```

---

## 🟢 Step 4 — Submit 3 Jobs (Different Priorities)

```
POST http://localhost:8080/v1/namespaces/production/queues/jobs/jobs

Job 1 — Priority 10 (Highest)
→ 202 Accepted
{
  "id":     "cc9eae40-bd72-4a1e-b1cb-d8283383ffc6",
  "type":   "echo",
  "status": "queued",
  "priority": 10,
  "payload": {"task":"send_welcome_email","user_id":1001}
}

Job 2 — Priority 5
→ 202 Accepted
{
  "id":     "8ebab164-5e07-4027-967a-604993581467",
  "type":   "echo",
  "status": "queued",
  "priority": 5,
  "payload": {"task":"generate_report","report_id":"RPT-2024"}
}

Job 3 — Priority 1 (Lowest)
→ 202 Accepted
{
  "id":     "9b28a53e-1547-4284-8f76-b8b71b309d4f",
  "type":   "echo",
  "status": "queued",
  "priority": 1,
  "payload": {"task":"sync_crm","customer":"Acme Corp"}
}
```

---

## 🟢 Step 5 — Worker Processes All 3 Jobs (Real Logs)

Worker output from `go run ./cmd/worker`:

```json
{"level":"INFO","msg":"Starting TaskForge — Worker Pool"}
{"level":"INFO","msg":"Starting Worker Leaser","queue_id":"f71bbf36...","worker_id":"worker-1782286235558868900"}

{"level":"INFO","msg":"Successfully leased job","job_id":"cc9eae40...","type":"echo"}
{"level":"INFO","msg":"EchoHandler executing job","job_id":"cc9eae40...","payload":"{\"task\": \"send_welcome_email\", \"user_id\": 1001}"}

{"level":"INFO","msg":"Successfully leased job","job_id":"8ebab164...","type":"echo"}
{"level":"INFO","msg":"EchoHandler executing job","job_id":"8ebab164...","payload":"{\"task\": \"generate_report\", \"report_id\": \"RPT-2024\"}"}

{"level":"INFO","msg":"Successfully leased job","job_id":"9b28a53e...","type":"echo"}
{"level":"INFO","msg":"EchoHandler executing job","job_id":"9b28a53e...","payload":"{\"task\": \"sync_crm\", \"customer\": \"Acme Corp\"}"}

{"level":"INFO","msg":"Job completed successfully","job_id":"cc9eae40...","duration_ms":0}
{"level":"INFO","msg":"Job completed successfully","job_id":"8ebab164...","duration_ms":0}
{"level":"INFO","msg":"Job completed successfully","job_id":"9b28a53e...","duration_ms":0}
```

---

## 🟢 Step 6 — PostgreSQL Database Verification

Direct query against the running Postgres container:

```sql
SELECT id, type, status, priority, attempt_count FROM jobs ORDER BY priority DESC;
```

| id | type | **status** | priority | attempt_count |
|----|------|-----------|----------|---------------|
| cc9eae40... | echo | **succeeded** | 10 | 1 |
| 8ebab164... | echo | **succeeded** | 5  | 1 |
| 9b28a53e... | echo | **succeeded** | 1  | 1 |

---

## 🟢 Step 7 — Job Attempts Audit Trail (Postgres)

```sql
SELECT job_id, attempt_no, worker_id, status FROM job_attempts;
```

| job_id | attempt_no | worker_id | status |
|--------|-----------|-----------|--------|
| cc9eae40... | 1 | worker-178228... | **succeeded** |
| 8ebab164... | 1 | worker-178228... | **succeeded** |
| 9b28a53e... | 1 | worker-178228... | **succeeded** |

> Each job was processed exactly **once**, by the same worker, with a full audit trail.

---

## 🟢 Step 8 — Timeline Events (Append-Only Audit Log)

```sql
SELECT event_type, message, created_at FROM timeline_events;
```

| event_type | message | created_at |
|------------|---------|------------|
| job_succeeded | Job completed successfully | 2026-06-24 07:30:35 |
| job_succeeded | Job completed successfully | 2026-06-24 07:30:35 |
| job_succeeded | Job completed successfully | 2026-06-24 07:30:35 |

---

## 🏁 Summary

| Component | Status |
|-----------|--------|
| PostgreSQL (Docker) | ✅ Running |
| Redis (Docker) | ✅ Running |
| TaskForge API (`cmd/api`) | ✅ Running on :8080 |
| TaskForge Worker (`cmd/worker`) | ✅ Running |
| Jobs submitted | ✅ 3 jobs queued via REST API |
| Jobs processed | ✅ 3/3 jobs `succeeded` |
| Audit trail | ✅ `job_attempts` + `timeline_events` populated |
| Concurrency control | ✅ Semaphore-bounded (limit: 10) |
| Priority ordering | ✅ Priority 10 leased before Priority 1 |
| Graceful shutdown | ✅ WaitGroup drain on SIGINT |
