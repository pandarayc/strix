package skill

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Skill is a knowledge module loaded on demand.
type Skill struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Body        string   `yaml:"-"` // content after YAML frontmatter
	Path        string   `yaml:"-"` // source file path
	Paths       []string `yaml:"paths"` // conditional: auto-activate when these path patterns match
	active      bool     // activated (either manual or auto)
}

// Registry holds all loaded skills.
type Registry struct {
	skills  map[string]*Skill
	pending map[string]*Skill // conditional skills not yet activated
}

// NewRegistry creates a new skill registry.
func NewRegistry() *Registry {
	return &Registry{
		skills:  make(map[string]*Skill),
		pending: make(map[string]*Skill),
	}
}

// Register adds a skill to the registry.
func (r *Registry) Register(s *Skill) {
	if len(s.Paths) > 0 {
		r.pending[s.Name] = s
	} else {
		s.active = true
		r.skills[s.Name] = s
	}
}

// Get returns a skill by name (checks active first, then pending).
func (r *Registry) Get(name string) (*Skill, bool) {
	s, ok := r.skills[name]
	if ok {
		return s, true
	}
	s, ok = r.pending[name]
	if ok {
		// Auto-activate on explicit load
		s.active = true
		r.skills[name] = s
		delete(r.pending, name)
	}
	return s, ok
}

// List returns all active skill names.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.skills))
	for name := range r.skills {
		names = append(names, name)
	}
	return names
}

// PendingCount returns the number of conditional skills not yet activated.
func (r *Registry) PendingCount() int {
	return len(r.pending)
}

// CheckAndActivate checks file paths against pending conditional skills.
// Returns skills that were newly activated.
func (r *Registry) CheckAndActivate(filePaths []string) []*Skill {
	var activated []*Skill
	for name, s := range r.pending {
		for _, filePath := range filePaths {
			if matchAnyPath(filePath, s.Paths) {
				s.active = true
				r.skills[name] = s
				delete(r.pending, name)
				activated = append(activated, s)
				break
			}
		}
	}
	return activated
}

func matchAnyPath(filePath string, patterns []string) bool {
	for _, p := range patterns {
		// Simple glob matching
		matched, err := filepath.Match(p, filePath)
		if err == nil && matched {
			return true
		}
		// Also match against the filename only
		matched, err = filepath.Match(p, filepath.Base(filePath))
		if err == nil && matched {
			return true
		}
	}
	return false
}

// Descriptions returns a lightweight listing for the system prompt.
func (r *Registry) Descriptions() string {
	if len(r.skills) == 0 {
		return "(no skills available)"
	}
	var b strings.Builder
	for _, s := range r.skills {
		b.WriteString(fmt.Sprintf("- %s: %s\n", s.Name, s.Description))
	}
	return b.String()
}

// LoadDir recursively scans dir for */SKILL.md files and loads them.
func (r *Registry) LoadDir(dir string) error {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && d.Name() == "SKILL.md" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("walk skills dir %s: %w", dir, err)
	}

	for _, path := range files {
		skill, err := ParseFile(path)
		if err != nil {
			continue
		}
		skill.Path = path
		r.Register(skill)
	}
	return nil
}

// ParseFile parses a SKILL.md file with YAML frontmatter.
func ParseFile(path string) (*Skill, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	if !scanner.Scan() {
		return nil, fmt.Errorf("empty file")
	}
	if strings.TrimSpace(scanner.Text()) != "---" {
		return nil, fmt.Errorf("expected --- frontmatter start")
	}

	var yamlLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		yamlLines = append(yamlLines, line)
	}

	var skill Skill
	if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), &skill); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}

	var bodyLines []string
	for scanner.Scan() {
		bodyLines = append(bodyLines, scanner.Text())
	}
	skill.Body = strings.TrimSpace(strings.Join(bodyLines, "\n"))

	if skill.Name == "" {
		skill.Name = filepath.Base(filepath.Dir(path))
	}

	return &skill, nil
}
