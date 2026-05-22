package lockfile

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	LockAPIVersion = "trove.io/v1"
	LockKind       = "TroveLock"
	DefaultLockFile = ".trove.lock.yaml"
)

type LockFile struct {
	APIVersion  string         `yaml:"apiVersion"`
	Kind        string         `yaml:"kind"`
	GeneratedBy *GeneratedInfo `yaml:"generatedBy,omitempty"`
	Project     ProjectInfo    `yaml:"project"`
	Installs    []InstallEntry `yaml:"installs"`
}

type GeneratedInfo struct {
	Name        string    `yaml:"name"`
	Version     string    `yaml:"version"`
	GeneratedAt time.Time `yaml:"generatedAt"`
}

type ProjectInfo struct {
	Org  string `yaml:"org"`
	Name string `yaml:"name"`
	Repo string `yaml:"repo,omitempty"`
}

type InstallEntry struct {
	Package           string           `yaml:"package"`
	RequestedSelector string           `yaml:"requestedSelector"`
	Version           string           `yaml:"version"`
	Digest            string           `yaml:"digest"`
	InstalledAt       time.Time        `yaml:"installedAt"`
	Artifacts         []ArtifactPin    `yaml:"artifacts"`
}

type ArtifactPin struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
	Digest string `yaml:"digest"`
}

func Parse(content []byte) (*LockFile, error) {
	var lock LockFile
	if err := yaml.Unmarshal(content, &lock); err != nil {
		return nil, fmt.Errorf("parse lock file: %w", err)
	}

	if lock.APIVersion == "" {
		lock.APIVersion = LockAPIVersion
	}
	if lock.Kind == "" {
		lock.Kind = LockKind
	}

	if err := Validate(&lock); err != nil {
		return nil, err
	}

	return &lock, nil
}

func ParseFile(path string) (*LockFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read lock file: %w", err)
	}
	return Parse(content)
}

func Validate(lock *LockFile) error {
	if lock.APIVersion != LockAPIVersion {
		return fmt.Errorf("unsupported apiVersion %q, expected %q", lock.APIVersion, LockAPIVersion)
	}
	if lock.Kind != LockKind {
		return fmt.Errorf("unsupported kind %q, expected %q", lock.Kind, LockKind)
	}
	if lock.Project.Org == "" {
		return fmt.Errorf("project.org is required")
	}
	if lock.Project.Name == "" {
		return fmt.Errorf("project.name is required")
	}

	seen := make(map[string]bool)
	for i, install := range lock.Installs {
		if install.Package == "" {
			return fmt.Errorf("installs[%d].package is required", i)
		}
		if install.Version == "" {
			return fmt.Errorf("installs[%d].version is required", i)
		}
		if install.Digest == "" {
			return fmt.Errorf("installs[%d].digest is required", i)
		}
		if seen[install.Package] {
			return fmt.Errorf("duplicate package %q in installs", install.Package)
		}
		seen[install.Package] = true

		for j, artifact := range install.Artifacts {
			if artifact.Source == "" {
				return fmt.Errorf("installs[%d].artifacts[%d].source is required", i, j)
			}
			if artifact.Target == "" {
				return fmt.Errorf("installs[%d].artifacts[%d].target is required", i, j)
			}
		}
	}

	return nil
}

func New(toolName, toolVersion string, project ProjectInfo) *LockFile {
	return &LockFile{
		APIVersion: LockAPIVersion,
		Kind:       LockKind,
		GeneratedBy: &GeneratedInfo{
			Name:        toolName,
			Version:     toolVersion,
			GeneratedAt: time.Now().UTC(),
		},
		Project: project,
	}
}

func (l *LockFile) AddInstall(entry InstallEntry) {
	for i, existing := range l.Installs {
		if existing.Package == entry.Package {
			l.Installs[i] = entry
			return
		}
	}
	l.Installs = append(l.Installs, entry)
}

func (l *LockFile) GetInstall(pkg string) *InstallEntry {
	for i := range l.Installs {
		if l.Installs[i].Package == pkg {
			return &l.Installs[i]
		}
	}
	return nil
}

func (l *LockFile) RemoveInstall(pkg string) {
	for i, entry := range l.Installs {
		if entry.Package == pkg {
			l.Installs = append(l.Installs[:i], l.Installs[i+1:]...)
			return
		}
	}
}

func (l *LockFile) Marshal() ([]byte, error) {
	return yaml.Marshal(l)
}

func (l *LockFile) WriteFile(path string) error {
	content, err := l.Marshal()
	if err != nil {
		return fmt.Errorf("marshal lock file: %w", err)
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("write lock file: %w", err)
	}
	return nil
}
