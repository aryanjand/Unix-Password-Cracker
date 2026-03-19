package sqlc

type UpsertWorkerStateParams struct {
	WorkerID  string
	State     string
	LastError string
}

type InsertWorkerFailureParams struct {
	WorkerID string
	Reason   string
}

type AssignTaskParams struct {
	ChunkID    uint64
	WorkerID   string
	ChunkStart uint64
	ChunkEnd   uint64
}

type CompleteTaskWithFoundParams struct {
	FoundPassword string
	ChunkID       uint64
}

type FailTaskParams struct {
	FailureReason string
	ChunkID       uint64
}

type InsertCheckpointParams struct {
	WorkerID      string
	ChunkID       uint64
	ChunkStart    uint64
	ChunkEnd      uint64
	DeltaTested   int64
	TotalTested   int64
	ThreadsActive int64
	CurrentRate   float64
	CurrentChunk  string
}
