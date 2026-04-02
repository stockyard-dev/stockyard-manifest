package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/stockyard-dev/stockyard-manifest/internal/store"
)

type Server struct {
	db  *store.DB
	mux *http.ServeMux
}

func New(db *store.DB) *Server {
	s := &Server{db: db, mux: http.NewServeMux()}

	s.mux.HandleFunc("GET /api/projects", s.listProjects)
	s.mux.HandleFunc("POST /api/projects", s.createProject)
	s.mux.HandleFunc("GET /api/projects/{id}", s.getProject)
	s.mux.HandleFunc("DELETE /api/projects/{id}", s.deleteProject)

	s.mux.HandleFunc("GET /api/projects/{id}/dependencies", s.listDeps)
	s.mux.HandleFunc("POST /api/projects/{id}/dependencies", s.addDep)
	s.mux.HandleFunc("POST /api/projects/{id}/import", s.importDeps)
	s.mux.HandleFunc("DELETE /api/dependencies/{id}", s.deleteDep)

	s.mux.HandleFunc("GET /api/projects/{id}/vulnerabilities", s.projectVulns)
	s.mux.HandleFunc("GET /api/dependencies/{id}/vulnerabilities", s.depVulns)
	s.mux.HandleFunc("POST /api/dependencies/{id}/vulnerabilities", s.addVuln)
	s.mux.HandleFunc("DELETE /api/vulnerabilities/{id}", s.deleteVuln)

	s.mux.HandleFunc("GET /api/projects/{id}/licenses", s.licenseSummary)
	s.mux.HandleFunc("GET /api/projects/{id}/sbom", s.exportSBOM)

	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)

	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/ui", http.StatusFound)
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"projects": orEmpty(s.db.ListProjects())})
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	var p store.Project
	json.NewDecoder(r.Body).Decode(&p)
	if p.Name == "" {
		writeErr(w, 400, "name required")
		return
	}
	s.db.CreateProject(&p)
	writeJSON(w, 201, s.db.GetProject(p.ID))
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request) {
	p := s.db.GetProject(r.PathValue("id"))
	if p == nil {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, p)
}

func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request) {
	s.db.DeleteProject(r.PathValue("id"))
	writeJSON(w, 200, map[string]string{"deleted": "ok"})
}

func (s *Server) listDeps(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"dependencies": orEmpty(s.db.ListDependencies(r.PathValue("id")))})
}

func (s *Server) addDep(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("id")
	if s.db.GetProject(pid) == nil {
		writeErr(w, 404, "project not found")
		return
	}
	var dep store.Dependency
	json.NewDecoder(r.Body).Decode(&dep)
	if dep.Name == "" {
		writeErr(w, 400, "name required")
		return
	}
	dep.ProjectID = pid
	if dep.LatestVersion != "" && dep.Version != dep.LatestVersion {
		dep.Outdated = true
	}
	s.db.AddDependency(&dep)
	writeJSON(w, 201, s.db.GetDependency(dep.ID))
}

func (s *Server) importDeps(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("id")
	if s.db.GetProject(pid) == nil {
		writeErr(w, 404, "project not found")
		return
	}
	var req struct {
		Dependencies []store.Dependency `json:"dependencies"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	count, err := s.db.ImportDependencies(pid, req.Dependencies)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]int{"imported": count})
}

func (s *Server) deleteDep(w http.ResponseWriter, r *http.Request) {
	s.db.DeleteDependency(r.PathValue("id"))
	writeJSON(w, 200, map[string]string{"deleted": "ok"})
}

func (s *Server) projectVulns(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"vulnerabilities": orEmpty(s.db.ProjectVulnerabilities(r.PathValue("id")))})
}

func (s *Server) depVulns(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"vulnerabilities": orEmpty(s.db.ListVulnerabilities(r.PathValue("id")))})
}

func (s *Server) addVuln(w http.ResponseWriter, r *http.Request) {
	depID := r.PathValue("id")
	if s.db.GetDependency(depID) == nil {
		writeErr(w, 404, "dependency not found")
		return
	}
	var v store.Vulnerability
	json.NewDecoder(r.Body).Decode(&v)
	if v.Title == "" {
		writeErr(w, 400, "title required")
		return
	}
	v.DependencyID = depID
	s.db.AddVulnerability(&v)
	writeJSON(w, 201, v)
}

func (s *Server) deleteVuln(w http.ResponseWriter, r *http.Request) {
	s.db.DeleteVulnerability(r.PathValue("id"))
	writeJSON(w, 200, map[string]string{"deleted": "ok"})
}

func (s *Server) licenseSummary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"licenses": orEmpty(s.db.LicenseSummary(r.PathValue("id")))})
}

func (s *Server) exportSBOM(w http.ResponseWriter, r *http.Request) {
	sbom := s.db.ExportSBOM(r.PathValue("id"))
	writeJSON(w, 200, map[string]any{"sbom": orEmpty(sbom)})
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.db.Stats()) }

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	st := s.db.Stats()
	writeJSON(w, 200, map[string]any{"status": "ok", "service": "manifest", "dependencies": st.Dependencies, "vulnerabilities": st.Vulnerabilities})
}

func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func init() { log.SetFlags(log.LstdFlags | log.Lshortfile) }
