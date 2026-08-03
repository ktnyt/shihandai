package tui

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ktnyt/shihandai/internal/drill"
	"github.com/ktnyt/shihandai/internal/naginata"
	"github.com/ktnyt/shihandai/internal/sentence"
)

func newTestModel() Model {
	engine := naginata.NewEngine(80 * time.Millisecond)
	d := drill.New(drill.DefaultConfig(), 1, nil)
	gen := &sentence.Generator{Cfg: sentence.DefaultConfig(), Rand: rand.New(rand.NewSource(1))}
	return New(engine, d, gen, "", "なし")
}

func TestViewWhileLoading(t *testing.T) {
	m := newTestModel()
	if !strings.Contains(m.View(), "生成中") {
		t.Errorf("生成中の表示がない:\n%s", m.View())
	}
}

func TestTypingFlow(t *testing.T) {
	m := newTestModel()

	// 例文が届いて練習が始まる
	next, _ := m.Update(lineMsg{level: 1, line: sentence.Line{
		Units:  []string{"あ", "い"},
		Source: "wordbank",
	}})
	m = next.(Model)

	view := m.View()
	for _, want := range []string{"あ", "つぎ", "[J]", "kpm"} {
		if !strings.Contains(view, want) {
			t.Errorf("表示に %q がない:\n%s", want, view)
		}
	}

	// 「あ」(J) を打ってウィンドウ経過を待つと1単位進む
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = next.(Model)
	next, _ = m.Update(tickMsg(time.Now().Add(time.Second)))
	m = next.(Model)

	if m.drill.Pos() != 1 {
		t.Fatalf("Pos() = %d, want 1", m.drill.Pos())
	}
	if !strings.Contains(m.View(), "[K]") {
		t.Errorf("次の運指ヒントが「い」[K] になっていない:\n%s", m.View())
	}
}

func TestQuitSavesWithoutPath(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("Esc で終了コマンドが返らない")
	}
}
