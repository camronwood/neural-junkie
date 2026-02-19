package repo

import "time"

// RepositoryIndex contains indexed information about a repository
type RepositoryIndex struct {
	Path            string                 `json:"path"`
	Name            string                 `json:"name"`
	LastIndexed     time.Time              `json:"last_indexed"`
	FileCount       int                    `json:"file_count"`
	TotalSize       int64                  `json:"total_size"`
	Structure       *DirectoryNode         `json:"structure"`
	KeyFiles        map[string]string      `json:"key_files"`    // filename -> content
	Dependencies    map[string][]string    `json:"dependencies"` // package manager -> dependencies
	GitInfo         *GitInfo               `json:"git_info"`
	ArchitectureDoc string                 `json:"architecture_doc"` // Generated architecture overview
	CodePatterns    []string               `json:"code_patterns"`    // Identified patterns/frameworks
	FileModTimes    map[string]time.Time   `json:"file_mod_times"`   // Track file modification times for change detection
	SourceFiles     map[string]*SourceFile `json:"source_files"`     // path -> compressed source file content
}

// DirectoryNode represents a directory in the repository structure
type DirectoryNode struct {
	Name        string           `json:"name"`
	Path        string           `json:"path"`
	IsDirectory bool             `json:"is_directory"`
	Size        int64            `json:"size,omitempty"`
	Children    []*DirectoryNode `json:"children,omitempty"`
	Language    string           `json:"language,omitempty"` // File language if not a directory
	Summary     string           `json:"summary,omitempty"`  // Brief description
}

// SourceFile represents a source code file with compressed content
type SourceFile struct {
	Path           string    `json:"path"`
	Language       string    `json:"language"`
	Size           int64     `json:"size"`              // Original size in bytes
	CompressedSize int64     `json:"compressed_size"`   // Compressed size in bytes
	Content        string    `json:"content"`           // gzipped + base64 encoded content
	ModTime        time.Time `json:"mod_time"`          // Last modification time
	Summary        string    `json:"summary,omitempty"` // Brief description (optional)
}

// GitInfo contains git repository information
type GitInfo struct {
	Branch         string       `json:"branch"`
	LastCommit     string       `json:"last_commit"`
	LastCommitMsg  string       `json:"last_commit_msg"`
	LastCommitDate time.Time    `json:"last_commit_date"`
	RecentCommits  []CommitInfo `json:"recent_commits"` // Last 10 commits
}

// CommitInfo represents a git commit
type CommitInfo struct {
	Hash    string    `json:"hash"`
	Author  string    `json:"author"`
	Date    time.Time `json:"date"`
	Message string    `json:"message"`
}

// KeyFileType defines important file types to extract
var KeyFileTypes = []string{
	"README.md",
	"README",
	"package.json",
	"go.mod",
	"Cargo.toml",
	"requirements.txt",
	"Pipfile",
	"pom.xml",
	"build.gradle",
	"Makefile",
	"Dockerfile",
	"docker-compose.yml",
	".env.example",
	"ARCHITECTURE.md",
	"CONTRIBUTING.md",
}

// LanguageExtensions maps file extensions to languages
var LanguageExtensions = map[string]string{
	".go":   "Go",
	".py":   "Python",
	".js":   "JavaScript",
	".ts":   "TypeScript",
	".tsx":  "TypeScript",
	".jsx":  "JavaScript",
	".java": "Java",
	".rs":   "Rust",
	".c":    "C",
	".cpp":  "C++",
	".cs":   "C#",
	".rb":   "Ruby",
	".php":  "PHP",
	".sh":   "Shell",
	".bash": "Bash",
	".yml":  "YAML",
	".yaml": "YAML",
	".json": "JSON",
	".xml":  "XML",
	".md":   "Markdown",
	".sql":  "SQL",
}
