package generator

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed all:templates
var templateFS embed.FS

// Config holds all the user's choices for template generation
type Config struct {
	ProjectName string

	// HTTP Frameworks
	IsEcho  bool
	IsFiber bool

	// Databases
	HasDB       bool
	HasPostgres bool
	HasMysql    bool
	HasSqlite   bool

	// WebSocket
	HasWebSocket bool

	// Features
	HasHTML bool

	// Task Runners
	HasMakefile bool
	HasTaskfile bool

	// Template-specific values
	HttpPort string
	DbUser   string
	DbPass   string
	DbName   string
	DbPort   string
}

func NewConfig(projectName string, options map[string]struct{}) Config {
	cfg := Config{ProjectName: projectName}

	_, cfg.IsEcho = options["Echo"]
	_, cfg.IsFiber = options["Fiber"]
	_, cfg.HasPostgres = options["PostgreSQL"]
	_, cfg.HasMysql = options["MySQL"]
	_, cfg.HasSqlite = options["SQLite"]
	_, cfg.HasWebSocket = options["gorilla/websocket"]
	_, cfg.HasHTML = options["Enable HTML templates"]
	_, cfg.HasMakefile = options["Makefile"]
	_, cfg.HasTaskfile = options["Taskfile"]

	cfg.HasDB = cfg.HasPostgres || cfg.HasMysql || cfg.HasSqlite

	// Default values for templates
	cfg.HttpPort = "8080"
	cfg.DbUser = "user"
	cfg.DbPass = "password"
	cfg.DbName = "mydatabase"
	cfg.DbPort = "5432"

	return cfg
}

// Generate creates the project structure and files
func Generate(cfg Config) error {
	projectPath, err := filepath.Abs(cfg.ProjectName)
	if err != nil {
		return fmt.Errorf("could not get absolute path for project: %w", err)
	}

	templateRoot := "templates"

	err = fs.WalkDir(templateFS, templateRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Determine destination path
		destPath := strings.Replace(path, templateRoot, projectPath, 1)
		destPath = strings.TrimSuffix(destPath, ".tmpl")

		// Conditional file generation
		if !cfg.HasDB && (strings.Contains(destPath, "postgres") || strings.Contains(destPath, "domain") || strings.Contains(destPath, "usecase")) {
			return nil
		}
		if !cfg.HasMakefile && strings.HasSuffix(destPath, "Makefile") {
			return nil
		}
		if !cfg.HasTaskfile && strings.HasSuffix(destPath, "Taskfile.yml") {
			return nil
		}
		if !cfg.HasHTML && strings.Contains(destPath, "web") {
			return nil
		}

		// Create destination directory
		if err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
			return err
		}

		// Parse and execute template from the embedded FS
		tmpl, err := template.ParseFS(templateFS, path)
		if err != nil {
			return fmt.Errorf("error parsing template %s: %w", path, err)
		}

		file, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("error creating file %s: %w", destPath, err)
		}
		defer file.Close()

		if err := tmpl.Execute(file, cfg); err != nil {
			return fmt.Errorf("error executing template %s: %w", path, err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Run go mod init
	cmd := exec.Command("go", "mod", "init", cfg.ProjectName)
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run 'go mod init': %w", err)
	}

	// Run go mod tidy
	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run 'go mod tidy': %w", err)
	}

	fmt.Printf("\nProject '%s' generated successfully!\n", cfg.ProjectName)

	return nil
}
