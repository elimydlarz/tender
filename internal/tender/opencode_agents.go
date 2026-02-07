package tender

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoverPrimaryAgents returns primary agents defined in local OpenCode config.
// Sources:
// - OPENCODE_CONFIG (if set)
// - opencode.json / opencode.jsonc in repo root
// - .opencode/agents/*.md (and OPENCODE_CONFIG_DIR/agents/*.md)
func DiscoverPrimaryAgents(root string) ([]string, error) {
	set := map[string]bool{}

	candidates := make([]string, 0)
	if custom := strings.TrimSpace(os.Getenv("OPENCODE_CONFIG")); custom != "" {
		candidates = append(candidates, custom)
	}
	candidates = append(candidates,
		filepath.Join(root, "opencode.json"),
		filepath.Join(root, "opencode.jsonc"),
	)

	for _, p := range candidates {
		if err := loadAgentsFromJSONConfig(p, set); err != nil {
			return nil, err
		}
	}

	agentDirs := []string{
		filepath.Join(root, ".opencode", "agents"),
	}
	if customDir := strings.TrimSpace(os.Getenv("OPENCODE_CONFIG_DIR")); customDir != "" {
		agentDirs = append(agentDirs, filepath.Join(customDir, "agents"))
	}

	for _, d := range agentDirs {
		if err := loadAgentsFromMarkdownDir(d, set); err != nil {
			return nil, err
		}
	}

	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

type openCodeConfig struct {
	DefaultAgent string                        `json:"default_agent"`
	Agent        map[string]openCodeAgentEntry `json:"agent"`
}

type openCodeAgentEntry struct {
	Mode    string `json:"mode"`
	Disable bool   `json:"disable"`
}

func loadAgentsFromJSONConfig(path string, set map[string]bool) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.IsDir() {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	clean := stripJSONCComments(string(data))

	var cfg openCodeConfig
	if err := json.Unmarshal([]byte(clean), &cfg); err != nil {
		return fmt.Errorf("failed parsing %s: %w", path, err)
	}

	if name := strings.TrimSpace(cfg.DefaultAgent); name != "" {
		set[name] = true
	}
	for name, entry := range cfg.Agent {
		if entry.Disable {
			continue
		}
		mode := strings.ToLower(strings.TrimSpace(entry.Mode))
		if mode == "subagent" {
			continue
		}
		set[name] = true
	}
	return nil
}

func loadAgentsFromMarkdownDir(dir string, set map[string]bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".md" && ext != ".markdown" {
			continue
		}
		p := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		mode, disabled := parseMarkdownAgentFrontmatter(string(data))
		if disabled {
			continue
		}
		if strings.ToLower(strings.TrimSpace(mode)) == "subagent" {
			continue
		}
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		if strings.TrimSpace(name) != "" {
			set[name] = true
		}
	}
	return nil
}

func parseMarkdownAgentFrontmatter(content string) (mode string, disabled bool) {
	text := strings.TrimLeft(content, "\ufeff\n\r\t ")
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		return "", false
	}
	lines := strings.Split(text, "\n")
	if len(lines) < 2 {
		return "", false
	}

	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "---" {
			break
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:idx]))
		val := strings.TrimSpace(line[idx+1:])
		val = strings.Trim(val, "\"'")
		switch key {
		case "mode":
			mode = val
		case "disable", "disabled":
			v := strings.ToLower(val)
			disabled = v == "true" || v == "yes" || v == "1"
		}
	}
	return mode, disabled
}

// stripJSONCComments removes // and /* */ comments while preserving string literals.
func stripJSONCComments(in string) string {
	var out strings.Builder
	out.Grow(len(in))

	inString := false
	escaped := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(in); i++ {
		c := in[i]
		next := byte(0)
		if i+1 < len(in) {
			next = in[i+1]
		}

		if inLineComment {
			if c == '\n' {
				inLineComment = false
				out.WriteByte(c)
			}
			continue
		}

		if inBlockComment {
			if c == '*' && next == '/' {
				inBlockComment = false
				i++
			}
			continue
		}

		if inString {
			out.WriteByte(c)
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}

		if c == '"' {
			inString = true
			out.WriteByte(c)
			continue
		}

		if c == '/' && next == '/' {
			inLineComment = true
			i++
			continue
		}
		if c == '/' && next == '*' {
			inBlockComment = true
			i++
			continue
		}

		out.WriteByte(c)
	}

	return out.String()
}
