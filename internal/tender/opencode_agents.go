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

// DiscoverPrimaryAgents returns primary agents from `opencode agent list`.
// No local config parsing fallback is used.
func DiscoverPrimaryAgents(root string) ([]string, error) {
	out, err := runOpenCode(root, "agent", "list")
	if err != nil {
		return nil, fmt.Errorf("opencode agent list failed: %w", err)
	}

	agents := parseOpenCodeAgentList(out)
	if len(agents) == 0 {
		return nil, fmt.Errorf("opencode agent list returned no usable agents")
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

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		first := strings.TrimSpace(fields[0])
		if strings.EqualFold(first, "name") || strings.EqualFold(first, "agent") {
			continue
		}
		if !agentNameRE.MatchString(first) {
			continue
		}
		set[first] = true
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
				if agentNameRE.MatchString(strings.TrimSpace(v)) {
					set[strings.TrimSpace(v)] = true
				}
			case map[string]interface{}:
				for _, k := range []string{"name", "agent", "id"} {
					if raw, ok := v[k]; ok {
						if s, ok := raw.(string); ok && agentNameRE.MatchString(strings.TrimSpace(s)) {
							set[strings.TrimSpace(s)] = true
							break
						}
					}
				}
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
						for _, k := range []string{"name", "agent", "id"} {
							if rawName, ok := v[k]; ok {
								if s, ok := rawName.(string); ok && agentNameRE.MatchString(strings.TrimSpace(s)) {
									set[strings.TrimSpace(s)] = true
									break
								}
							}
						}
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
