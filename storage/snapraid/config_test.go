package snapraid

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	cfg := Config{
		Parity:  []string{"/mnt/parity/snapraid.parity"},
		Data:    map[string]string{"d1": "/mnt/d1"},
		Content: []string{"/mnt/d1/.snapraid.content"},
	}
	out := Render(cfg)
	for _, expected := range []string{"parity /mnt/parity/snapraid.parity", "data d1 /mnt/d1", "content /mnt/d1/.snapraid.content"} {
		if !strings.Contains(out, expected) {
			t.Fatalf("missing %q in %q", expected, out)
		}
	}
}
