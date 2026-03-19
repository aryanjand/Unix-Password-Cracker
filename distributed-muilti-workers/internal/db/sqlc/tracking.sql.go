package sqlc

import "context"

const upsertWorkerState = `-- name: UpsertWorkerState :exec
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
    updated_at = UTC_TIMESTAMP()
`

func (q *Queries) UpsertWorkerState(ctx context.Context, arg UpsertWorkerStateParams) error {
	_, err := q.db.ExecContext(ctx, upsertWorkerState, arg.WorkerID, arg.State, arg.LastError)
	return err
}

const insertWorkerFailure = `-- name: InsertWorkerFailure :exec
INSERT INTO worker_failures (worker_id, reason)
VALUES (?, ?)
`

func (q *Queries) InsertWorkerFailure(ctx context.Context, arg InsertWorkerFailureParams) error {
	_, err := q.db.ExecContext(ctx, insertWorkerFailure, arg.WorkerID, arg.Reason)
	return err
}

const assignTask = `-- name: AssignTask :exec
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
    updated_at = UTC_TIMESTAMP()
`

func (q *Queries) AssignTask(ctx context.Context, arg AssignTaskParams) error {
	_, err := q.db.ExecContext(ctx, assignTask, arg.ChunkID, arg.WorkerID, arg.ChunkStart, arg.ChunkEnd)
	return err
}

const completeTask = `-- name: CompleteTask :exec
UPDATE tasks
SET
    status = 'complete',
    completed_at = UTC_TIMESTAMP(),
    updated_at = UTC_TIMESTAMP()
WHERE chunk_id = ?
`

func (q *Queries) CompleteTask(ctx context.Context, chunkID uint64) error {
	_, err := q.db.ExecContext(ctx, completeTask, chunkID)
	return err
}

const completeTaskWithFound = `-- name: CompleteTaskWithFound :exec
UPDATE tasks
SET
    status = 'found',
    found_password = ?,
    completed_at = UTC_TIMESTAMP(),
    updated_at = UTC_TIMESTAMP()
WHERE chunk_id = ?
`

func (q *Queries) CompleteTaskWithFound(ctx context.Context, arg CompleteTaskWithFoundParams) error {
	_, err := q.db.ExecContext(ctx, completeTaskWithFound, arg.FoundPassword, arg.ChunkID)
	return err
}

const failTask = `-- name: FailTask :exec
UPDATE tasks
SET
    status = 'failed',
    failure_reason = ?,
    completed_at = UTC_TIMESTAMP(),
    updated_at = UTC_TIMESTAMP()
WHERE chunk_id = ?
`

func (q *Queries) FailTask(ctx context.Context, arg FailTaskParams) error {
	_, err := q.db.ExecContext(ctx, failTask, arg.FailureReason, arg.ChunkID)
	return err
}

const insertCheckpoint = `-- name: InsertCheckpoint :exec
INSERT INTO worker_checkpoints (
    worker_id,
    chunk_id,
    chunk_start,
    chunk_end,
    delta_tested,
    total_tested,
    threads_active,
    current_rate,
    current_chunk
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?
)
`

func (q *Queries) InsertCheckpoint(ctx context.Context, arg InsertCheckpointParams) error {
	_, err := q.db.ExecContext(
		ctx,
		insertCheckpoint,
		arg.WorkerID,
		arg.ChunkID,
		arg.ChunkStart,
		arg.ChunkEnd,
		arg.DeltaTested,
		arg.TotalTested,
		arg.ThreadsActive,
		arg.CurrentRate,
		arg.CurrentChunk,
	)
	return err
}
