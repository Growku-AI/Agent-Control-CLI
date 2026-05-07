package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type ProjectGoalResponse struct {
	ID          string  `json:"id"`
	ProjectID   string  `json:"project_id"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	TargetDate  *string `json:"target_date"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type CreateProjectGoalRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
	TargetDate  *string `json:"target_date"`
	Status      *string `json:"status"`
}

type UpdateProjectGoalRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	TargetDate  *string `json:"target_date"`
	Status      *string `json:"status"`
}

func projectGoalToResponse(goal db.ProjectGoal) ProjectGoalResponse {
	return ProjectGoalResponse{
		ID:          uuidToString(goal.ID),
		ProjectID:   uuidToString(goal.ProjectID),
		Title:       goal.Title,
		Description: textToPtr(goal.Description),
		TargetDate:  timestampToPtr(goal.TargetDate),
		Status:      goal.Status,
		CreatedAt:   timestampToString(goal.CreatedAt),
		UpdatedAt:   timestampToString(goal.UpdatedAt),
	}
}

func parseOptionalTimestamp(value *string) (pgtype.Timestamptz, error) {
	if value == nil {
		return pgtype.Timestamptz{}, nil
	}
	t, err := time.Parse(time.RFC3339, *value)
	if err != nil {
		return pgtype.Timestamptz{}, err
	}
	return pgtype.Timestamptz{Time: t, Valid: true}, nil
}

func (h *Handler) ListProjectGoals(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	workspaceID := h.resolveWorkspaceID(r)
	if _, err := h.Queries.GetProjectInWorkspace(r.Context(), db.GetProjectInWorkspaceParams{ID: parseUUID(projectID), WorkspaceID: parseUUID(workspaceID)}); err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	goals, err := h.Queries.ListProjectGoals(r.Context(), parseUUID(projectID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list goals")
		return
	}

	resp := make([]ProjectGoalResponse, len(goals))
	for i, goal := range goals {
		resp[i] = projectGoalToResponse(goal)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateProjectGoal(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	workspaceID := h.resolveWorkspaceID(r)
	if _, err := h.Queries.GetProjectInWorkspace(r.Context(), db.GetProjectInWorkspaceParams{ID: parseUUID(projectID), WorkspaceID: parseUUID(workspaceID)}); err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var req CreateProjectGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	targetDate, err := parseOptionalTimestamp(req.TargetDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "target_date must be RFC3339")
		return
	}

	status := pgtype.Text{}
	if req.Status != nil {
		status = pgtype.Text{String: *req.Status, Valid: true}
	}

	goal, err := h.Queries.CreateProjectGoal(r.Context(), db.CreateProjectGoalParams{
		ProjectID:   parseUUID(projectID),
		Title:       req.Title,
		Description: ptrToText(req.Description),
		TargetDate:  targetDate,
		Status:      status,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create goal")
		return
	}

	writeJSON(w, http.StatusCreated, projectGoalToResponse(goal))
}

func (h *Handler) GetProjectGoal(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	goalID := chi.URLParam(r, "goalId")

	goal, err := h.Queries.GetProjectGoal(r.Context(), db.GetProjectGoalParams{ID: parseUUID(goalID), ProjectID: parseUUID(projectID)})
	if err != nil {
		writeError(w, http.StatusNotFound, "goal not found")
		return
	}

	writeJSON(w, http.StatusOK, projectGoalToResponse(goal))
}

func (h *Handler) UpdateProjectGoal(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	goalID := chi.URLParam(r, "goalId")

	if _, err := h.Queries.GetProjectGoal(r.Context(), db.GetProjectGoalParams{ID: parseUUID(goalID), ProjectID: parseUUID(projectID)}); err != nil {
		writeError(w, http.StatusNotFound, "goal not found")
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req UpdateProjectGoalRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var rawFields map[string]json.RawMessage
	json.Unmarshal(bodyBytes, &rawFields)

	params := db.UpdateProjectGoalParams{ID: parseUUID(goalID), ProjectID: parseUUID(projectID)}
	if req.Title != nil {
		params.Title = pgtype.Text{String: *req.Title, Valid: true}
	}
	if _, ok := rawFields["description"]; ok {
		params.Description = ptrToText(req.Description)
	}
	if _, ok := rawFields["target_date"]; ok {
		targetDate, err := parseOptionalTimestamp(req.TargetDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "target_date must be RFC3339")
			return
		}
		params.TargetDate = targetDate
	}
	if req.Status != nil {
		params.Status = pgtype.Text{String: *req.Status, Valid: true}
	}

	goal, err := h.Queries.UpdateProjectGoal(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update goal")
		return
	}

	writeJSON(w, http.StatusOK, projectGoalToResponse(goal))
}

func (h *Handler) DeleteProjectGoal(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	goalID := chi.URLParam(r, "goalId")

	affected, err := h.Queries.DeleteProjectGoal(r.Context(), db.DeleteProjectGoalParams{ID: parseUUID(goalID), ProjectID: parseUUID(projectID)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete goal")
		return
	}
	if affected == 0 {
		writeError(w, http.StatusNotFound, "goal not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
