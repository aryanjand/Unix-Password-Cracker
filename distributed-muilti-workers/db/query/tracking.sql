-- name: UpsertWorkerState :exec
INSERT INTO workers (
    worker_id,
    state,
    last_error,
    last_heartbeat_at
) VALUES (
    ?, ?, NULLIF(?, ''), UTC_TIMESTAMP()
)
ON DUPLICATE KEY UPDATE
    state = VALUES(state),
    last_error = VALUES(last_error),
    last_heartbeat_at = VALUES(last_heartbeat_at),
    updated_at = UTC_TIMESTAMP();

-- name: InsertWorkerFailure :exec
INSERT INTO worker_failures (worker_id, reason)
VALUES (?, ?);

-- name: AssignTask :exec
INSERT INTO tasks (
    chunk_id,
    worker_id,
    chunk_start,
    chunk_end,
    status,
    assigned_at
) VALUES (
    ?, ?, ?, ?, 'assigned', UTC_TIMESTAMP()
)
ON DUPLICATE KEY UPDATE
    worker_id = VALUES(worker_id),
    chunk_start = VALUES(chunk_start),
    chunk_end = VALUES(chunk_end),
    status = 'assigned',
    failure_reason = NULL,
    found_password = NULL,
    assigned_at = UTC_TIMESTAMP(),
    completed_at = NULL,
    updated_at = UTC_TIMESTAMP();

-- name: CompleteTask :exec
UPDATE tasks
SET
    status = 'complete',
    completed_at = UTC_TIMESTAMP(),
    updated_at = UTC_TIMESTAMP()
WHERE chunk_id = ?;

-- name: CompleteTaskWithFound :exec
UPDATE tasks
SET
    status = 'found',
    found_password = ?,
    completed_at = UTC_TIMESTAMP(),
    updated_at = UTC_TIMESTAMP()
WHERE chunk_id = ?;

-- name: FailTask :exec
UPDATE tasks
SET
    status = 'failed',
    failure_reason = ?,
    completed_at = UTC_TIMESTAMP(),
    updated_at = UTC_TIMESTAMP()
WHERE chunk_id = ?;

-- name: InsertCheckpoint :exec
INSERT INTO worker_checkpoints (
    worker_id,
    chunk_id,
    chunk_start,
    chunk_end,
    completed
) VALUES (
    ?, ?, ?, ?, ?
);
