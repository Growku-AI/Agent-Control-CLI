-- name: ListProjectGoals :many
SELECT * FROM project_goal
WHERE project_id = $1
ORDER BY created_at DESC;

-- name: GetProjectGoal :one
SELECT * FROM project_goal
WHERE id = $1 AND project_id = $2;

-- name: CreateProjectGoal :one
INSERT INTO project_goal (
    project_id,
    title,
    description,
    target_date,
    status
) VALUES (
    $1,
    $2,
    $3,
    $4,
    COALESCE(sqlc.narg('status')::text, 'active')
)
RETURNING *;

-- name: UpdateProjectGoal :one
UPDATE project_goal
SET
    title = COALESCE(sqlc.narg('title')::text, title),
    description = sqlc.narg('description')::text,
    target_date = sqlc.narg('target_date')::timestamptz,
    status = COALESCE(sqlc.narg('status')::text, status),
    updated_at = now()
WHERE id = $1 AND project_id = $2
RETURNING *;

-- name: DeleteProjectGoal :execrows
DELETE FROM project_goal
WHERE id = $1 AND project_id = $2;
