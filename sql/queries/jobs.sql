-- name: CreateJob :one
INSERT INTO jobs (
    job_type, status, start_time, end_time, result, cancel_requested
) VALUES ( $1, $2, $3, $4, $5, $6 )
RETURNING *;

-- name: GetJobById :one
SELECT * FROM jobs
WHERE id = $1 LIMIT 1;

-- name: UpdateJob :one
UPDATE jobs
SET status = $1, end_time = $2, result = $3, cancel_requested = $4, updated_at = CURRENT_TIMESTAMP
WHERE id = $5
RETURNING *;

-- name: GetCancelRequested :one
SELECT cancel_requested FROM jobs WHERE id = $1;

-- name: UpdateCancelRequested :one
UPDATE jobs SET cancel_requested = $1, updated_at = CURRENT_TIMESTAMP  
WHERE id = $2
RETURNING *;

-- name: GetRunningJobsByType :many
SELECT * FROM jobs 
WHERE job_type = $1 AND status = 1;


-- name: GetLatestJobByType :one
SELECT * FROM jobs
WHERE job_type = $1
ORDER BY created_at DESC
LIMIT 1;

