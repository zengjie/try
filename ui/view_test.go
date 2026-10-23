package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderPreviewLineTruncation(t *testing.T) {
	// CJK characters are double-width; the rendered line must never exceed the
	// terminal width regardless of content.
	cases := []struct {
		name    string
		preview string
		width   int
	}{
		{"ascii fits", "short preview", 80},
		{"ascii overflow", "this preview is definitely longer than the terminal", 30},
		{"cjk overflow", "用于测试消费者组的实验目录，包含完整的客户端代码", 24},
		{"mixed overflow", "redis 流实验 with consumer groups 消费者组", 20},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := renderPreviewLine(c.preview, c.width)
			if w := lipgloss.Width(got); w > c.width {
				t.Errorf("rendered width %d exceeds terminal width %d: %q", w, c.width, got)
			}
		})
	}

	// Degenerate width should not panic and returns nothing to draw.
	if got := renderPreviewLine("anything", 2); got != "" {
		t.Errorf("degenerate width should render empty, got %q", got)
	}
}

func TestRenderTitleBarNarrowWidth(t *testing.T) {
	// Before the first WindowSizeMsg the width is 0; narrow terminals are
	// smaller than the title. Neither may panic.
	for _, w := range []int{0, 1, 10, 30} {
		_ = renderTitleBar(w)
	}
	for _, w := range []int{60, 80, 121} {
		if got := lipgloss.Width(renderTitleBar(w)); got != w {
			t.Errorf("title bar width = %d, want %d", got, w)
		}
	}
}

func TestViewZeroWidthNoPanic(t *testing.T) {
	m := NewModel()
	_ = m.View()
}
