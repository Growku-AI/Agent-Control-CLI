package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type ProjectRepositoryResponse struct {
	ID            string  `json:"id"`
	ProjectID     string  `json:"project_id"`
	URL           string  `json:"url"`
	Name          string  `json:"name"`
	DefaultBranch *string `json:"default_branch"`
	CreatedAt     string  `json:"created_at"`
}

type AddProjectRepositoryRequest struct {
	URL           string  `json:"url"`
	Name          string  `json:"name"`
	DefaultBranch *string `json:"default_branch"`
}

func projectRepositoryToResponse(repo db.ProjectRepository) ProjectRepositoryResponse {
	return ProjectRepositoryResponse{
		ID:            uuidToString(repo.ID),
		ProjectID:     uuidToString(repo.ProjectID),
		URL:           repo.Url,
		Name:          repo.Name,
		DefaultBranch: textToPtr(repo.DefaultBranch),
		CreatedAt:     timestampToString(repo.CreatedAt),
	}
}

func (h *Handler) ListProjectRepositories(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	workspaceID := h.resolveWorkspaceID(r)
	if _, err := h.Queries.GetProjectInWorkspace(r.Context(), db.GetProjectInWorkspaceParams{ID: parseUUID(projectID), WorkspaceID: parseUUID(workspaceID)}); err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	repos, err := h.Queries.ListProjectRepositories(r.Context(), parseUUID(projectID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list repositories")
		return
	}

	resp := make([]ProjectRepositoryResponse, len(repos))
	for i, repo := range repos {
		resp[i] = projectRepositoryToResponse(repo)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) AddProjectRepository(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	workspaceID := h.resolveWorkspaceID(r)
	if _, err := h.Queries.GetProjectInWorkspace(r.Context(), db.GetProjectInWorkspaceParams{ID: parseUUID(projectID), WorkspaceID: parseUUID(workspaceID)}); err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var req AddProjectRepositoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	defaultBranch := pgtype.Text{}
	if req.DefaultBranch != nil {
		defaultBranch = pgtype.Text{String: *req.DefaultBranch, Valid: true}
	}

	repo, err := h.Queries.AddProjectRepository(r.Context(), db.AddProjectRepositoryParams{
		ProjectID:     parseUUID(projectID),
		Url:           req.URL,
		Name:          req.Name,
		DefaultBranch: defaultBranch,
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "repository already exists for this project")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to add repository")
		return
	}

	writeJSON(w, http.StatusCreated, projectRepositoryToResponse(repo))
}

func (h *Handler) RemoveProjectRepository(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	repoID := chi.URLParam(r, "repoId")

	affected, err := h.Queries.RemoveProjectRepository(r.Context(), db.RemoveProjectRepositoryParams{ID: parseUUID(repoID), ProjectID: parseUUID(projectID)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove repository")
		return
	}
	if affected == 0 {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
