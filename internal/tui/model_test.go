package tui

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ktnyt/shihandai/internal/drill"
	"github.com/ktnyt/shihandai/internal/lesson"
	"github.com/ktnyt/shihandai/internal/naginata"
)

func newTestModel(t *testing.T) Model {
	t.Helper()
	engine := naginata.NewEngine(80 * time.Millisecond)
	d := drill.New(drill.DefaultConfig(), 1, nil)
	gen := lesson.NewGenerator(lesson.DefaultConfig(), rand.New(rand.NewSource(1)))
	m, err := New(engine, d, gen, "")
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// pressChord は1単位分の同時押しを送り、ウィンドウ経過の tick で確定させる。
func pressChord(m Model, chord naginata.KeySet) Model {
	for _, key := range chord.Keys() {
		var msg tea.KeyMsg
		if key == naginata.KeySpace {
			msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
		} else {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(strings.ToLower(key.Label()))}
		}
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	next, _ := m.Update(tickMsg(time.Now().Add(200 * time.Millisecond)))
	return next.(Model)
}

func TestNewStartsWithTypableLine(t *testing.T) {
	m := newTestModel(t)
	line := m.drill.Line()
	if len(line) == 0 {
		t.Fatal("最初の行が空")
	}
	allowed := m.drill.Allowed()
	for _, u := range line {
		if _, ok := lesson.Segment(u, allowed); !ok {
			t.Errorf("使えないかな %q が行に含まれる", u)
		}
	}

	view := m.View()
	for _, want := range []string{"使えるかな", "つぎ", "kpm"} {
		if !strings.Contains(view, want) {
			t.Errorf("表示に %q がない:\n%s", want, view)
		}
	}
}

func TestTypingWholeLinePromotesAndStartsNext(t *testing.T) {
	m := newTestModel(t)
	line := m.drill.Line()

	// 生成された行を正しい打鍵でノーミス高速で打ち切る
	for _, u := range line {
		chord, ok := naginata.ChordFor(u)
		if !ok {
			t.Fatalf("%q の打鍵が見つからない", u)
		}
		m = pressChord(m, chord)
	}

	if m.drill.Level != 2 {
		t.Fatalf("Level = %d, want 2 (ノーミス高速で昇格するはず)", m.drill.Level)
	}
	if len(m.drill.Line()) == 0 || m.drill.Pos() != 0 {
		t.Fatalf("次の行が始まっていない: line=%v pos=%d", m.drill.Line(), m.drill.Pos())
	}
	if !strings.Contains(m.View(), "新しいかなを追加") {
		t.Errorf("昇格メッセージが出ていない:\n%s", m.View())
	}
}

func TestWrongInputFlashesError(t *testing.T) {
	m := newTestModel(t)
	expected := m.drill.Expected()

	// 期待と違うかなを1つ打つ
	wrongKey := naginata.KeyJ // あ
	if expected == "あ" {
		wrongKey = naginata.KeyK // い
	}
	m = pressChord(m, naginata.Set(wrongKey))

	if m.drill.LineErrors() != 1 {
		t.Fatalf("LineErrors = %d, want 1", m.drill.LineErrors())
	}
	if !strings.Contains(m.View(), "ミス") {
		t.Errorf("ミス表示がない:\n%s", m.View())
	}
	if m.drill.Pos() != 0 {
		t.Errorf("ミスで位置が進んだ: pos=%d", m.drill.Pos())
	}
}

func TestQuitOnEsc(t *testing.T) {
	m := newTestModel(t)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("Esc で終了コマンドが返らない")
	}
}
