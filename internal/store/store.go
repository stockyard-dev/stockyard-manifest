package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct{ db *sql.DB }

type Project struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Ecosystem     string `json:"ecosystem,omitempty"` // npm, go, pip, cargo, maven
	RepoURL       string `json:"repo_url,omitempty"`
	CreatedAt     string `json:"created_at"`
	DepCount      int    `json:"dep_count"`
	VulnCount     int    `json:"vuln_count"`
	OutdatedCount int    `json:"outdated_count"`
	LastScan      string `json:"last_scan,omitempty"`
}

type Dependency struct {
	ID            string `json:"id"`
	ProjectID     string `json:"project_id"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	LatestVersion string `json:"latest_version,omitempty"`
	License       string `json:"license,omitempty"`
	Ecosystem     string `json:"ecosystem,omitempty"`
	Direct        bool   `json:"direct"`
	Outdated      bool   `json:"outdated"`
	Deprecated    bool   `json:"deprecated"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	VulnCount     int    `json:"vuln_count"`
}

type Vulnerability struct {
	ID           string `json:"id"`
	DependencyID string `json:"dependency_id"`
	CVEID        string `json:"cve_id"`
	Severity     string `json:"severity"` // critical, high, medium, low
	Title        string `json:"title"`
	Description  string `json:"description,omitempty"`
	FixVersion   string `json:"fix_version,omitempty"`
	URL          string `json:"url,omitempty"`
	CreatedAt    string `json:"created_at"`
}

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	dsn := filepath.Join(dataDir, "manifest.db") + "?_journal_mode=WAL&_busy_timeout=5000"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY, name TEXT NOT NULL, ecosystem TEXT DEFAULT '',
			repo_url TEXT DEFAULT '', created_at TEXT DEFAULT (datetime('now')),
			last_scan TEXT DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS dependencies (
			id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id),
			name TEXT NOT NULL, version TEXT DEFAULT '', latest_version TEXT DEFAULT '',
			license TEXT DEFAULT '', ecosystem TEXT DEFAULT '',
			direct INTEGER DEFAULT 1, outdated INTEGER DEFAULT 0, deprecated INTEGER DEFAULT 0,
			created_at TEXT DEFAULT (datetime('now')), updated_at TEXT DEFAULT (datetime('now')),
			UNIQUE(project_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS vulnerabilities (
			id TEXT PRIMARY KEY, dependency_id TEXT NOT NULL REFERENCES dependencies(id) ON DELETE CASCADE,
			cve_id TEXT DEFAULT '', severity TEXT DEFAULT 'medium',
			title TEXT NOT NULL, description TEXT DEFAULT '',
			fix_version TEXT DEFAULT '', url TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_deps_project ON dependencies(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_vulns_dep ON vulnerabilities(dependency_id)`,
	} {
		if _, err := db.Exec(q); err != nil {
			return nil, fmt.Errorf("migrate: %w", err)
		}
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS extras(resource TEXT NOT NULL,record_id TEXT NOT NULL,data TEXT NOT NULL DEFAULT '{}',PRIMARY KEY(resource, record_id))`)
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }
func genID() string        { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string          { return time.Now().UTC().Format(time.RFC3339) }

// ── Projects ──

func (d *DB) hydrateProject(p *Project) {
	d.db.QueryRow(`SELECT COUNT(*) FROM dependencies WHERE project_id=?`, p.ID).Scan(&p.DepCount)
	d.db.QueryRow(`SELECT COUNT(*) FROM dependencies WHERE project_id=? AND outdated=1`, p.ID).Scan(&p.OutdatedCount)
	d.db.QueryRow(`SELECT COUNT(*) FROM vulnerabilities WHERE dependency_id IN (SELECT id FROM dependencies WHERE project_id=?)`, p.ID).Scan(&p.VulnCount)
}

func (d *DB) CreateProject(p *Project) error {
	p.ID = genID()
	p.CreatedAt = now()
	_, err := d.db.Exec(`INSERT INTO projects (id,name,ecosystem,repo_url,created_at) VALUES (?,?,?,?,?)`,
		p.ID, p.Name, p.Ecosystem, p.RepoURL, p.CreatedAt)
	return err
}

func (d *DB) GetProject(id string) *Project {
	var p Project
	if err := d.db.QueryRow(`SELECT id,name,ecosystem,repo_url,created_at,last_scan FROM projects WHERE id=?`, id).Scan(
		&p.ID, &p.Name, &p.Ecosystem, &p.RepoURL, &p.CreatedAt, &p.LastScan); err != nil {
		return nil
	}
	d.hydrateProject(&p)
	return &p
}

func (d *DB) ListProjects() []Project {
	rows, err := d.db.Query(`SELECT id,name,ecosystem,repo_url,created_at,last_scan FROM projects ORDER BY name`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Project
	for rows.Next() {
		var p Project
		rows.Scan(&p.ID, &p.Name, &p.Ecosystem, &p.RepoURL, &p.CreatedAt, &p.LastScan)
		d.hydrateProject(&p)
		out = append(out, p)
	}
	return out
}

func (d *DB) DeleteProject(id string) error {
	d.db.Exec(`DELETE FROM vulnerabilities WHERE dependency_id IN (SELECT id FROM dependencies WHERE project_id=?)`, id)
	d.db.Exec(`DELETE FROM dependencies WHERE project_id=?`, id)
	_, err := d.db.Exec(`DELETE FROM projects WHERE id=?`, id)
	return err
}

// ── Dependencies ──

func (d *DB) AddDependency(dep *Dependency) error {
	// Upsert: if name+project already exists, update version
	var existingID string
	err := d.db.QueryRow(`SELECT id FROM dependencies WHERE project_id=? AND name=?`, dep.ProjectID, dep.Name).Scan(&existingID)
	t := now()
	if err == sql.ErrNoRows {
		dep.ID = genID()
		dep.CreatedAt = t
		dep.UpdatedAt = t
		direct := 1
		if !dep.Direct {
			direct = 0
		}
		outdated := 0
		if dep.Outdated {
			outdated = 1
		}
		deprecated := 0
		if dep.Deprecated {
			deprecated = 1
		}
		_, err := d.db.Exec(`INSERT INTO dependencies (id,project_id,name,version,latest_version,license,ecosystem,direct,outdated,deprecated,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
			dep.ID, dep.ProjectID, dep.Name, dep.Version, dep.LatestVersion, dep.License, dep.Ecosystem, direct, outdated, deprecated, dep.CreatedAt, dep.UpdatedAt)
		return err
	}
	dep.ID = existingID
	outdated := 0
	if dep.LatestVersion != "" && dep.Version != dep.LatestVersion {
		outdated = 1
		dep.Outdated = true
	}
	deprecated := 0
	if dep.Deprecated {
		deprecated = 1
	}
	direct := 1
	if !dep.Direct {
		direct = 0
	}
	_, err = d.db.Exec(`UPDATE dependencies SET version=?,latest_version=?,license=?,ecosystem=?,direct=?,outdated=?,deprecated=?,updated_at=? WHERE id=?`,
		dep.Version, dep.LatestVersion, dep.License, dep.Ecosystem, direct, outdated, deprecated, t, existingID)
	// Update project last_scan
	d.db.Exec(`UPDATE projects SET last_scan=? WHERE id=?`, t, dep.ProjectID)
	return err
}

func (d *DB) hydrateDep(dep *Dependency) {
	d.db.QueryRow(`SELECT COUNT(*) FROM vulnerabilities WHERE dependency_id=?`, dep.ID).Scan(&dep.VulnCount)
}

func (d *DB) ListDependencies(projectID string) []Dependency {
	rows, err := d.db.Query(`SELECT id,project_id,name,version,latest_version,license,ecosystem,direct,outdated,deprecated,created_at,updated_at FROM dependencies WHERE project_id=? ORDER BY name`, projectID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Dependency
	for rows.Next() {
		var dep Dependency
		var direct, outdated, deprecated int
		rows.Scan(&dep.ID, &dep.ProjectID, &dep.Name, &dep.Version, &dep.LatestVersion, &dep.License, &dep.Ecosystem, &direct, &outdated, &deprecated, &dep.CreatedAt, &dep.UpdatedAt)
		dep.Direct = direct == 1
		dep.Outdated = outdated == 1
		dep.Deprecated = deprecated == 1
		d.hydrateDep(&dep)
		out = append(out, dep)
	}
	return out
}

func (d *DB) GetDependency(id string) *Dependency {
	var dep Dependency
	var direct, outdated, deprecated int
	if err := d.db.QueryRow(`SELECT id,project_id,name,version,latest_version,license,ecosystem,direct,outdated,deprecated,created_at,updated_at FROM dependencies WHERE id=?`, id).Scan(
		&dep.ID, &dep.ProjectID, &dep.Name, &dep.Version, &dep.LatestVersion, &dep.License, &dep.Ecosystem, &direct, &outdated, &deprecated, &dep.CreatedAt, &dep.UpdatedAt); err != nil {
		return nil
	}
	dep.Direct = direct == 1
	dep.Outdated = outdated == 1
	dep.Deprecated = deprecated == 1
	d.hydrateDep(&dep)
	return &dep
}

func (d *DB) DeleteDependency(id string) error {
	d.db.Exec(`DELETE FROM vulnerabilities WHERE dependency_id=?`, id)
	_, err := d.db.Exec(`DELETE FROM dependencies WHERE id=?`, id)
	return err
}

// ── Bulk import ──

func (d *DB) ImportDependencies(projectID string, deps []Dependency) (int, error) {
	count := 0
	for i := range deps {
		deps[i].ProjectID = projectID
		if err := d.AddDependency(&deps[i]); err == nil {
			count++
		}
	}
	d.db.Exec(`UPDATE projects SET last_scan=? WHERE id=?`, now(), projectID)
	return count, nil
}

// ── Vulnerabilities ──

func (d *DB) AddVulnerability(v *Vulnerability) error {
	v.ID = genID()
	v.CreatedAt = now()
	if v.Severity == "" {
		v.Severity = "medium"
	}
	_, err := d.db.Exec(`INSERT INTO vulnerabilities (id,dependency_id,cve_id,severity,title,description,fix_version,url,created_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		v.ID, v.DependencyID, v.CVEID, v.Severity, v.Title, v.Description, v.FixVersion, v.URL, v.CreatedAt)
	return err
}

func (d *DB) ListVulnerabilities(depID string) []Vulnerability {
	q := `SELECT id,dependency_id,cve_id,severity,title,description,fix_version,url,created_at FROM vulnerabilities`
	var args []any
	if depID != "" {
		q += ` WHERE dependency_id=?`
		args = append(args, depID)
	}
	q += ` ORDER BY CASE severity WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 END, created_at DESC`
	rows, err := d.db.Query(q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Vulnerability
	for rows.Next() {
		var v Vulnerability
		rows.Scan(&v.ID, &v.DependencyID, &v.CVEID, &v.Severity, &v.Title, &v.Description, &v.FixVersion, &v.URL, &v.CreatedAt)
		out = append(out, v)
	}
	return out
}

func (d *DB) ProjectVulnerabilities(projectID string) []Vulnerability {
	rows, err := d.db.Query(`SELECT v.id,v.dependency_id,v.cve_id,v.severity,v.title,v.description,v.fix_version,v.url,v.created_at FROM vulnerabilities v JOIN dependencies dep ON v.dependency_id=dep.id WHERE dep.project_id=? ORDER BY CASE v.severity WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 END`, projectID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Vulnerability
	for rows.Next() {
		var v Vulnerability
		rows.Scan(&v.ID, &v.DependencyID, &v.CVEID, &v.Severity, &v.Title, &v.Description, &v.FixVersion, &v.URL, &v.CreatedAt)
		out = append(out, v)
	}
	return out
}

func (d *DB) DeleteVulnerability(id string) error {
	_, err := d.db.Exec(`DELETE FROM vulnerabilities WHERE id=?`, id)
	return err
}

// ── License analysis ──

type LicenseInfo struct {
	License string `json:"license"`
	Count   int    `json:"count"`
}

func (d *DB) LicenseSummary(projectID string) []LicenseInfo {
	rows, err := d.db.Query(`SELECT COALESCE(NULLIF(license,''),'unknown'), COUNT(*) FROM dependencies WHERE project_id=? GROUP BY license ORDER BY COUNT(*) DESC`, projectID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []LicenseInfo
	for rows.Next() {
		var l LicenseInfo
		rows.Scan(&l.License, &l.Count)
		out = append(out, l)
	}
	return out
}

// ── SBOM export ──

type SBOMEntry struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	License   string `json:"license"`
	Ecosystem string `json:"ecosystem"`
	Direct    bool   `json:"direct"`
}

func (d *DB) ExportSBOM(projectID string) []SBOMEntry {
	deps := d.ListDependencies(projectID)
	var out []SBOMEntry
	for _, dep := range deps {
		out = append(out, SBOMEntry{
			Name: dep.Name, Version: dep.Version,
			License: dep.License, Ecosystem: dep.Ecosystem, Direct: dep.Direct,
		})
	}
	return out
}

// ── Stats ──

type Stats struct {
	Projects        int            `json:"projects"`
	Dependencies    int            `json:"dependencies"`
	Vulnerabilities int            `json:"vulnerabilities"`
	Outdated        int            `json:"outdated"`
	BySeverity      map[string]int `json:"by_severity"`
	Ecosystems      []string       `json:"ecosystems"`
}

func (d *DB) Stats() Stats {
	var s Stats
	d.db.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&s.Projects)
	d.db.QueryRow(`SELECT COUNT(*) FROM dependencies`).Scan(&s.Dependencies)
	d.db.QueryRow(`SELECT COUNT(*) FROM vulnerabilities`).Scan(&s.Vulnerabilities)
	d.db.QueryRow(`SELECT COUNT(*) FROM dependencies WHERE outdated=1`).Scan(&s.Outdated)
	s.BySeverity = map[string]int{}
	rows, _ := d.db.Query(`SELECT severity, COUNT(*) FROM vulnerabilities GROUP BY severity`)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var sev string
			var c int
			rows.Scan(&sev, &c)
			s.BySeverity[sev] = c
		}
	}
	erows, _ := d.db.Query(`SELECT DISTINCT ecosystem FROM dependencies WHERE ecosystem != '' ORDER BY ecosystem`)
	if erows != nil {
		defer erows.Close()
		for erows.Next() {
			var e string
			erows.Scan(&e)
			s.Ecosystems = append(s.Ecosystems, e)
		}
	}
	if s.Ecosystems == nil {
		s.Ecosystems = []string{}
	}
	_ = strings.Join(nil, "") // avoid unused import
	return s
}

// ─── Extras: generic key-value storage for personalization custom fields ───

func (d *DB) GetExtras(resource, recordID string) string {
	var data string
	err := d.db.QueryRow(
		`SELECT data FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	).Scan(&data)
	if err != nil || data == "" {
		return "{}"
	}
	return data
}

func (d *DB) SetExtras(resource, recordID, data string) error {
	if data == "" {
		data = "{}"
	}
	_, err := d.db.Exec(
		`INSERT INTO extras(resource, record_id, data) VALUES(?, ?, ?)
		 ON CONFLICT(resource, record_id) DO UPDATE SET data=excluded.data`,
		resource, recordID, data,
	)
	return err
}

func (d *DB) DeleteExtras(resource, recordID string) error {
	_, err := d.db.Exec(
		`DELETE FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	)
	return err
}

func (d *DB) AllExtras(resource string) map[string]string {
	out := make(map[string]string)
	rows, _ := d.db.Query(
		`SELECT record_id, data FROM extras WHERE resource=?`,
		resource,
	)
	if rows == nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, data string
		rows.Scan(&id, &data)
		out[id] = data
	}
	return out
}
