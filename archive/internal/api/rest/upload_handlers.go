package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/timot/Quant_WebWork_GO/internal/auth"
	"github.com/timot/Quant_WebWork_GO/internal/config"
	"github.com/timot/Quant_WebWork_GO/internal/monitoring"
	"github.com/timot/Quant_WebWork_GO/pkg/models"
)

const (
	maxUploadSize = 1024 * 1024 * 1024 // 1GB
	uploadsDir    = "./uploads"
)

type ProjectUploadRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	Config      map[string]string `json:"config"`
}

// UploadHandler handles project uploads and deployments
type UploadHandler struct {
	cfg           *config.Config
	securityMon   *monitoring.SecurityMonitor
	projectStore  *models.ProjectStore
}

// NewUploadHandler creates a new upload handler
func NewUploadHandler(cfg *config.Config, securityMon *monitoring.SecurityMonitor, store *models.ProjectStore) *UploadHandler {
	return &UploadHandler{
		cfg:           cfg,
		securityMon:   securityMon,
		projectStore:  store,
	}
}

// HandleProjectUpload handles the upload of new projects
func (h *UploadHandler) HandleProjectUpload(w http.ResponseWriter, r *http.Request) {
	// Verify user is authenticated
	userID := auth.GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	// Parse project metadata
	var req ProjectUploadRequest
	if err := json.NewDecoder(strings.NewReader(r.FormValue("metadata"))).Decode(&req); err != nil {
		http.Error(w, "Invalid project metadata", http.StatusBadRequest)
		return
	}

	// Get uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type
	if !h.isValidFileType(header.Filename) {
		http.Error(w, "Invalid file type", http.StatusBadRequest)
		return
	}

	// Create project directory
	projectDir := filepath.Join(uploadsDir, req.Name)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		http.Error(w, "Error creating project directory", http.StatusInternalServerError)
		return
	}

	// Save file
	dst, err := os.Create(filepath.Join(projectDir, header.Filename))
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Error copying file", http.StatusInternalServerError)
		return
	}

	// Create project record
	project := &models.Project{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Config:      req.Config,
		FilePath:    filepath.Join(projectDir, header.Filename),
		UserID:      userID,
	}

	// Perform security scan
	scanResult := h.securityMon.ScanProject(project)
	if !scanResult.IsSecure {
		// Clean up uploaded files
		os.RemoveAll(projectDir)
		http.Error(w, fmt.Sprintf("Security scan failed: %s", scanResult.Reason), http.StatusBadRequest)
		return
	}

	// Save project to store
	if err := h.projectStore.Create(project); err != nil {
		os.RemoveAll(projectDir)
		http.Error(w, "Error saving project", http.StatusInternalServerError)
		return
	}

	// Deploy project
	if err := h.deployProject(project); err != nil {
		http.Error(w, fmt.Sprintf("Error deploying project: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(project)
}

// HandleProjectList returns a list of all projects
func (h *UploadHandler) HandleProjectList(w http.ResponseWriter, r *http.Request) {
	projects, err := h.projectStore.List()
	if err != nil {
		http.Error(w, "Error retrieving projects", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(projects)
}

// HandleProjectDelete handles project deletion
func (h *UploadHandler) HandleProjectDelete(w http.ResponseWriter, r *http.Request) {
	// Verify user is authenticated
	userID := auth.GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	projectName := r.URL.Query().Get("name")
	if projectName == "" {
		http.Error(w, "Project name required", http.StatusBadRequest)
		return
	}

	// Get project
	project, err := h.projectStore.Get(projectName)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	// Verify ownership
	if project.UserID != userID {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Delete project files
	if err := os.RemoveAll(filepath.Dir(project.FilePath)); err != nil {
		http.Error(w, "Error deleting project files", http.StatusInternalServerError)
		return
	}

	// Delete from store
	if err := h.projectStore.Delete(projectName); err != nil {
		http.Error(w, "Error deleting project record", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Helper methods

func (h *UploadHandler) isValidFileType(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validTypes := map[string]bool{
		".zip": true,
		".tar": true,
		".gz":  true,
	}
	return validTypes[ext]
}

func (h *UploadHandler) deployProject(project *models.Project) error {
	// Implement native Go deployment logic here
	// This could include:
	// 1. Extracting archives
	// 2. Validating project structure
	// 3. Setting up runtime environment
	// 4. Starting necessary services
	return nil
}
