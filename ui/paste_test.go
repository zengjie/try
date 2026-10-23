package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func pasteMsg(text string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text), Paste: true}
}

func TestInsertableTextAcceptsPaste(t *testing.T) {
	url := "https://github.com/zengjie/try.git"
	if got := insertableText(pasteMsg(url)); got != url {
		t.Fatalf("paste dropped: got %q, want %q", got, url)
	}
}

func TestInsertableTextSingleRuneAndSpace(t *testing.T) {
	if got := insertableText(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}); got != "a" {
		t.Fatalf("single rune: got %q", got)
	}
	if got := insertableText(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}); got != " " {
		t.Fatalf("space: got %q", got)
	}
}

func TestInsertableTextRejectsNonText(t *testing.T) {
	cases := []tea.KeyMsg{
		{Type: tea.KeyEnter},
		{Type: tea.KeyCtrlG},
		{Type: tea.KeyUp},
		{Type: tea.KeyRunes, Runes: []rune("b"), Alt: true},
	}
	for _, msg := range cases {
		if got := insertableText(msg); got != "" {
			t.Errorf("%v should not be text, got %q", msg, got)
		}
	}
}

func TestInsertableTextStripsLineBreaks(t *testing.T) {
	if got := insertableText(pasteMsg("git@host:a.git\ngit@host:b.git")); got != "git@host:a.git git@host:b.git" {
		t.Fatalf("got %q", got)
	}
}

func TestCloneInputAcceptsPastedURL(t *testing.T) {
	url := "git@github.com:zengjie/try.git"
	m := NewModel()
	m.cloning = true

	updated, _ := m.Update(pasteMsg(url))
	got := updated.(Model).cloneInput
	if got != url {
		t.Fatalf("cloneInput = %q, want %q", got, url)
	}
}

func TestWorktreeAndDeleteInputAcceptPaste(t *testing.T) {
	m := NewModel()
	m.creatingWorktree = true
	updated, _ := m.Update(pasteMsg("my-branch"))
	if got := updated.(Model).worktreeInput; got != "my-branch" {
		t.Fatalf("worktreeInput = %q", got)
	}
}

func TestQueryAcceptsPaste(t *testing.T) {
	m := NewModel()
	updated, _ := m.Update(pasteMsg("proj"))
	if got := updated.(Model).query; got != "proj" {
		t.Fatalf("query = %q, want %q", got, "proj")
	}
}

func TestBackspaceDeletesWholeRune(t *testing.T) {
	m := NewModel()
	m.cloning = true
	updated, _ := m.Update(pasteMsg("测试"))
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if got := updated.(Model).cloneInput; got != "测" {
		t.Fatalf("cloneInput = %q, want %q", got, "测")
	}
}

func TestQueryBackspaceDeletesWholeRune(t *testing.T) {
	m := NewModel()
	m.SetQuery("测试")
	m.DeleteFromQuery()
	if m.query != "测" {
		t.Fatalf("query = %q, want %q", m.query, "测")
	}
}
