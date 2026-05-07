-- name: ListProjectRepositories :many
SELECT * FROM project_repository
WHERE project_id = $1
ORDER BY created_at DESC;

-- name: AddProjectRepository :one
INSERT INTO project_repository (
    project_id,
    url,
    name,
    default_branch
) VALUES (
    $1,
    $2,
    $3,
    COALESCE(sqlc.narg('default_branch')::text, 'main')
)
RETURNING *;

-- name: RemoveProjectRepository :execrows
DELETE FROM project_repository
WHERE id = $1 AND project_id = $2;
