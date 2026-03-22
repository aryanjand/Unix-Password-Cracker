CREATE TABLE IF NOT EXISTS workers (
    worker_id VARCHAR(255) PRIMARY KEY,
    state VARCHAR(32) NOT NULL,
    last_error TEXT NULL,
    last_heartbeat_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS worker_failures (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    worker_id VARCHAR(255) NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_worker_failures_worker_id_created (worker_id, created_at),
    CONSTRAINT fk_worker_failures_worker
        FOREIGN KEY (worker_id) REFERENCES workers(worker_id)
        ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tasks (
    chunk_id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
    worker_id VARCHAR(255) NOT NULL,
    chunk_start BIGINT UNSIGNED NOT NULL,
    chunk_end BIGINT UNSIGNED NOT NULL,
    status VARCHAR(32) NOT NULL,
    failure_reason TEXT NULL,
    found_password VARCHAR(255) NULL,
    assigned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_tasks_worker_id (worker_id),
    INDEX idx_tasks_status (status),
    CONSTRAINT fk_tasks_worker
        FOREIGN KEY (worker_id) REFERENCES workers(worker_id)
        ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS worker_checkpoints (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    worker_id VARCHAR(255) NOT NULL,
    chunk_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
    chunk_start BIGINT UNSIGNED NOT NULL DEFAULT 0,
    chunk_end BIGINT UNSIGNED NOT NULL DEFAULT 0,
    completed BIGINT UNSIGNED NOT NULL,
    reported_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_checkpoints_worker_reported (worker_id, reported_at),
    INDEX idx_checkpoints_chunk_id (chunk_id),
    CONSTRAINT fk_checkpoints_worker
        FOREIGN KEY (worker_id) REFERENCES workers(worker_id)
        ON DELETE CASCADE
);
