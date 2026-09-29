package metrics

import (
	"strings"
	"testing"
)

func TestRenderEscapesBoundedServiceLabel(t *testing.T) {
	text := Render([]Service{{Name: "api\"x", Healthy: true}})
	if !strings.Contains(text, "service=\"api\\\"x\"") {
		t.Fatal(text)
	}
}
