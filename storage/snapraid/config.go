package snapraid

import (
	"fmt"
	"strings"
)

type Config struct {
	Parity  []string
	Data    map[string]string
	Content []string
	Exclude []string
}

func Render(cfg Config) string {
	lines := make([]string, 0)
	for i, p := range cfg.Parity {
		if i == 0 {
			lines = append(lines, fmt.Sprintf("parity %s", p))
		} else {
			lines = append(lines, fmt.Sprintf("%d-parity %s", i+1, p))
		}
	}
	for label, path := range cfg.Data {
		lines = append(lines, fmt.Sprintf("data %s %s", label, path))
	}
	for _, c := range cfg.Content {
		lines = append(lines, fmt.Sprintf("content %s", c))
	}
	for _, e := range cfg.Exclude {
		lines = append(lines, fmt.Sprintf("exclude %s", e))
	}
	return strings.Join(lines, "\n") + "\n"
}
