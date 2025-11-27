package server

import (
	"encoding/json"
	"net/http"
)

// listProjects handles GET /project
// Returns a list of all projects (currently just the current project).
func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	dir := getDirectory(r.Context())
	var projects interface{}
	var err error

	if dir != "" {
		projects, err = s.projectService.ListForDir(r.Context(), dir)
	} else {
		projects, err = s.projectService.List(r.Context())
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

// getCurrentProject handles GET /project/current
// Returns the current project based on the working directory.
func (s *Server) getCurrentProject(w http.ResponseWriter, r *http.Request) {
	dir := getDirectory(r.Context())
	var project interface{}
	var err error

	if dir != "" {
		project, err = s.projectService.CurrentForDir(r.Context(), dir)
	} else {
		project, err = s.projectService.Current(r.Context())
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}
