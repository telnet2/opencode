package server

import (
	"net/http"

	"github.com/opencode-ai/opencode/pkg/types"
)

// listProjects handles GET /project
func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	directory := r.URL.Query().Get("directory")

	projects, err := s.sessionService.ListProjects(r.Context(), directory)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	// Ensure we return an empty array [] instead of null
	if projects == nil {
		projects = []*types.Project{}
	}

	writeJSON(w, http.StatusOK, projects)
}

// getCurrentProject handles GET /project/current
func (s *Server) getCurrentProject(w http.ResponseWriter, r *http.Request) {
	directory := getDirectory(r.Context())
	if directory == "" {
		directory = r.URL.Query().Get("directory")
	}

	if directory == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "directory is required")
		return
	}

	project, err := s.sessionService.GetCurrentProject(r.Context(), directory)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, project)
}
