package tender

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)
var agentNameRE = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
var lineAgentModeRE = regexp.MustCompile(`^([A-Za-z0-9._-]+)\s+\(([^)]+)\)\s*$`)

var systemAgentNames = map[string]bool{
	"build":      true,
	"summary":    true,
	"title":      true,
	"plan":       true,
	"compaction": true,
	"explore":    true,
	"general":    true,
}

// DiscoverPrimaryAgents returns custom primary agents from `opencode agent list`.
// No local config parsing fallback is used.
func DiscoverPrimaryAgents(root string) ([]string, error) {
	out, err := runOpenCode(root, "agent", "list")
	if err != nil {
		return nil, fmt.Errorf("opencode agent list failed: %w", err)
	}

	agents := parseOpenCodeAgentList(out)
	if len(agents) == 0 {
		return nil, fmt.Errorf("opencode agent list returned no custom primary agents")
	}

	sort.Strings(agents)
	return agents, nil
}

func runOpenCode(root string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Dir = root
	cmd.Stderr = io.Discard
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func parseOpenCodeAgentList(out string) []string {
	text := strings.TrimSpace(ansiRE.ReplaceAllString(out, ""))
	if text == "" {
		return nil
	}

	if strings.HasPrefix(text, "[") || strings.HasPrefix(text, "{") {
		if fromJSON := parseOpenCodeAgentListJSON(text); len(fromJSON) > 0 {
			sort.Strings(fromJSON)
			return fromJSON
		}
	}

	set := map[string]bool{}
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "opencode agent list") ||
			strings.HasPrefix(lower, "list all available agents") ||
			strings.HasPrefix(lower, "options:") ||
			strings.HasPrefix(lower, "error ") ||
			strings.HasPrefix(lower, "error:") {
			continue
		}
		name, mode, ok := parseTextAgentLine(line)
		if !ok {
			continue
		}
		if shouldSkipAgent(name, mode) {
			continue
		}
		set[name] = true
	}

	outList := make([]string, 0, len(set))
	for name := range set {
		outList = append(outList, name)
	}
	sort.Strings(outList)
	return outList
}

func parseOpenCodeAgentListJSON(text string) []string {
	set := map[string]bool{}

	var arr []interface{}
	if err := json.Unmarshal([]byte(text), &arr); err == nil {
		for _, item := range arr {
			switch v := item.(type) {
			case string:
				name := strings.TrimSpace(v)
				if !agentNameRE.MatchString(name) || shouldSkipAgent(name, "") {
					continue
				}
				set[name] = true
			case map[string]interface{}:
				name, mode, ok := parseJSONAgentRecord(v)
				if !ok || shouldSkipAgent(name, mode) {
					continue
				}
				set[name] = true
			}
		}
	}

	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(text), &obj); err == nil {
		for _, key := range []string{"agents", "items", "data"} {
			raw, ok := obj[key]
			if !ok {
				continue
			}
			if arrLike, ok := raw.([]interface{}); ok {
				for _, item := range arrLike {
					if v, ok := item.(map[string]interface{}); ok {
						name, mode, ok := parseJSONAgentRecord(v)
						if !ok || shouldSkipAgent(name, mode) {
							continue
						}
						set[name] = true
					}
				}
			}
		}
	}

	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func parseTextAgentLine(line string) (name string, mode string, ok bool) {
	if line == "" {
		return "", "", false
	}

	if m := lineAgentModeRE.FindStringSubmatch(line); len(m) == 3 {
		n := strings.TrimSpace(m[1])
		md := normalizeMode(m[2])
		if strings.EqualFold(n, "name") || strings.EqualFold(n, "agent") {
			return "", "", false
		}
		if !agentNameRE.MatchString(n) {
			return "", "", false
		}
		return n, md, true
	}

	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", "", false
	}
	n := strings.TrimSpace(fields[0])
	if strings.EqualFold(n, "name") || strings.EqualFold(n, "agent") {
		return "", "", false
	}
	if !agentNameRE.MatchString(n) {
		return "", "", false
	}

	md := ""
	if len(fields) > 1 {
		md = normalizeMode(fields[1])
	}
	return n, md, true
}

func parseJSONAgentRecord(v map[string]interface{}) (name string, mode string, ok bool) {
	var rawName string
	for _, k := range []string{"name", "agent", "id"} {
		if raw, has := v[k]; has {
			if s, ok := raw.(string); ok {
				rawName = strings.TrimSpace(s)
				if rawName != "" {
					break
				}
			}
		}
	}
	if rawName == "" || !agentNameRE.MatchString(rawName) {
		return "", "", false
	}

	rawMode := ""
	if raw, has := v["mode"]; has {
		if s, ok := raw.(string); ok {
			rawMode = s
		}
	}
	return rawName, normalizeMode(rawMode), true
}

func normalizeMode(raw string) string {
	md := strings.TrimSpace(strings.ToLower(raw))
	md = strings.Trim(md, "()[]{}")
	return md
}

func shouldSkipAgent(name string, mode string) bool {
	if strings.TrimSpace(name) == "" {
		return true
	}
	md := normalizeMode(mode)
	if md != "" && md != "primary" {
		return true
	}
	return IsSystemAgent(name)
}

func IsSystemAgent(name string) bool {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return false
	}
	return systemAgentNames[strings.ToLower(trimmedName)]
}
