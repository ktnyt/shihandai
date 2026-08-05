package tui

import (
	"math/rand"
	"slices"
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
	// テストではインターバルなしで即座に次の単語を出す
	m, err := New(engine, d, gen, "", 0)
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

// typeWord は現在の単語を正しい打鍵で打ち切る。
func typeWord(t *testing.T, m Model) Model {
	t.Helper()
	for _, u := range m.drill.Word() {
		chord, ok := naginata.ChordFor(u)
		if !ok {
			t.Fatalf("%q の打鍵が見つからない", u)
		}
		m = pressChord(m, chord)
	}
	return m
}

func TestNewStartsWithTypableWord(t *testing.T) {
	m := newTestModel(t)
	word := m.drill.Word()
	if len(word) == 0 {
		t.Fatal("最初の単語が空")
	}
	allowed := m.drill.Allowed()
	for _, u := range word {
		if _, ok := lesson.Segment(u, allowed); !ok {
			t.Errorf("使えないかな %q が単語に含まれる", u)
		}
	}

	view := m.View()
	for _, want := range []string{"使えるかな", "つぎ", "kps", "ミス率", "レベルアップ"} {
		if !strings.Contains(view, want) {
			t.Errorf("表示に %q がない:\n%s", want, view)
		}
	}
}

func TestTypingWordRecordsSuccessAndStartsNext(t *testing.T) {
	m := newTestModel(t)
	first := strings.Join(m.drill.Word(), "")

	m = typeWord(t, m)

	successes, total := m.drill.SuccessCount()
	if successes != 1 || total != 1 {
		t.Fatalf("SuccessCount = %d/%d, want 1/1", successes, total)
	}
	if len(m.drill.Word()) == 0 || m.drill.Pos() != 0 {
		t.Fatalf("次の単語が始まっていない: word=%v pos=%d", m.drill.Word(), m.drill.Pos())
	}
	if !strings.Contains(m.View(), "成功") {
		t.Errorf("成功メッセージが出ていない:\n%s", m.View())
	}
	_ = first
}

func TestSlowWordRecordsFailure(t *testing.T) {
	m := newTestModel(t)

	// しきい値を必ず超えるよう、経過時間を進めてから打ち切る
	m.drill.StartWord(m.drill.Word(), time.Now().Add(-time.Minute))
	m = typeWord(t, m)

	successes, total := m.drill.SuccessCount()
	if successes != 0 || total != 1 {
		t.Fatalf("SuccessCount = %d/%d, want 0/1", successes, total)
	}
	if !strings.Contains(m.View(), "時間超過") {
		t.Errorf("時間超過メッセージが出ていない:\n%s", m.View())
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

	if m.drill.WordErrors() != 1 {
		t.Fatalf("WordErrors = %d, want 1", m.drill.WordErrors())
	}
	if !strings.Contains(m.View(), "ミス") {
		t.Errorf("ミス表示がない:\n%s", m.View())
	}
	if m.drill.Pos() != 0 {
		t.Errorf("ミスで位置が進んだ: pos=%d", m.drill.Pos())
	}
}

func TestPromoteAfterWindowFilledWithSuccess(t *testing.T) {
	engine := naginata.NewEngine(80 * time.Millisecond)
	cfg := drill.DefaultConfig()
	cfg.WindowSize = 5
	cfg.MinNewKanaWords = 0
	d := drill.New(cfg, 1, nil)
	gen := lesson.NewGenerator(lesson.DefaultConfig(), rand.New(rand.NewSource(1)))
	m, err := New(engine, d, gen, "", 0)
	if err != nil {
		t.Fatal(err)
	}

	for range cfg.WindowSize {
		m = typeWord(t, m)
	}
	if m.drill.Level != 2 {
		t.Fatalf("Level = %d, want 2 (窓が成功で埋まったら昇格)", m.drill.Level)
	}

	// レベルアップ画面が出る。レベル1→2はかな追加ではなく長さの解放
	view := m.View()
	for _, want := range []string{"レベルアップ", "ながさ", "Space"} {
		if !strings.Contains(view, want) {
			t.Errorf("レベルアップ画面に %q がない:\n%s", want, view)
		}
	}

	// かなキーは無視される
	before := m.drill.Word()
	m = pressChord(m, naginata.Set(naginata.KeyJ))
	if !slices.Equal(m.drill.Word(), before) || !m.leveledUp {
		t.Fatal("レベルアップ画面でかなキーが処理された")
	}

	// Space で新しいレベルの単語が始まる
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	m = next.(Model)
	if m.leveledUp {
		t.Fatal("Space でレベルアップ画面が閉じない")
	}
	if len(m.drill.Word()) == 0 || m.drill.Pos() != 0 {
		t.Fatalf("次の単語が始まっていない: word=%v pos=%d", m.drill.Word(), m.drill.Pos())
	}
	// 再開後は普通に打てる
	m = typeWord(t, m)
	if _, total := m.drill.SuccessCount(); total != 1 {
		t.Errorf("レベルアップ後の単語が判定されていない")
	}
}

func TestEscOnLevelUpScreenQuits(t *testing.T) {
	engine := naginata.NewEngine(80 * time.Millisecond)
	cfg := drill.DefaultConfig()
	cfg.WindowSize = 2
	cfg.MinNewKanaWords = 0
	d := drill.New(cfg, 1, nil)
	gen := lesson.NewGenerator(lesson.DefaultConfig(), rand.New(rand.NewSource(1)))
	m, err := New(engine, d, gen, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	for range cfg.WindowSize {
		m = typeWord(t, m)
	}
	if !m.leveledUp {
		t.Fatal("前提が崩れた: レベルアップ画面になっていない")
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("レベルアップ画面の Esc で終了コマンドが返らない")
	}
}

func TestEscPausesAndHidesWord(t *testing.T) {
	m := newTestModel(t)
	word := strings.Join(m.drill.Word(), "")
	before := m.View()

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)
	if cmd != nil {
		t.Fatal("1回目の Esc で終了してはいけない")
	}

	view := m.View()
	if !strings.Contains(view, "一時停止中") {
		t.Errorf("一時停止の表示がない:\n%s", view)
	}
	// 単語と運指ヒントは隠れ、伏せ字が出る
	if strings.Contains(view, "つぎ") {
		t.Errorf("一時停止中に運指ヒントが見えている:\n%s", view)
	}
	if len(word) > 1 && strings.Contains(view, word) {
		t.Errorf("一時停止中に単語 %q が見えている:\n%s", word, view)
	}
	if !strings.Contains(view, "●") {
		t.Errorf("伏せ字が出ていない:\n%s", view)
	}
	// 統計は見えたまま、行数も変わらない（レイアウトがずれない）
	for _, want := range []string{"kps", "ミス率", "レベルアップ"} {
		if !strings.Contains(view, want) {
			t.Errorf("一時停止中に %q が消えている:\n%s", want, view)
		}
	}
	if got, want := strings.Count(view, "\n"), strings.Count(before, "\n"); got != want {
		t.Errorf("一時停止で行数が変わった: %d → %d\n通常:\n%s\n停止中:\n%s", want, got, before, view)
	}
}

func TestKeysIgnoredWhilePaused(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)

	m = pressChord(m, naginata.Set(naginata.KeyJ))
	if m.drill.Pos() != 0 || m.drill.WordErrors() != 0 {
		t.Fatalf("一時停止中の打鍵が判定された: pos=%d errors=%d",
			m.drill.Pos(), m.drill.WordErrors())
	}
}

func TestSpaceResumesSameWordWithFreshTimer(t *testing.T) {
	m := newTestModel(t)
	word := m.drill.Word()

	// 1文字打ってから一時停止し、しばらく経ってから再開する
	chord, _ := naginata.ChordFor(word[0])
	m = pressChord(m, chord)
	if m.drill.Pos() != 1 {
		t.Fatalf("前提が崩れた: pos=%d", m.drill.Pos())
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	m = next.(Model)

	if m.paused {
		t.Fatal("Space で再開していない")
	}
	if !slices.Equal(m.drill.Word(), word) {
		t.Errorf("再開後の単語が変わった: %v → %v", word, m.drill.Word())
	}
	if m.drill.Pos() != 0 {
		t.Errorf("再開後は最初から打ち直すべき: pos=%d", m.drill.Pos())
	}
	if e := m.drill.Elapsed(time.Now()); e > time.Second {
		t.Errorf("再開時に計測がやり直されていない: elapsed=%v", e)
	}
	// 再開後は普通に打てる
	m = typeWord(t, m)
	if _, total := m.drill.SuccessCount(); total != 1 {
		t.Errorf("再開後の単語が判定されていない")
	}
}

func TestEscTwiceQuits(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("一時停止中の Esc で終了コマンドが返らない")
	}
}

func TestCtrlCQuitsImmediately(t *testing.T) {
	m := newTestModel(t)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("Ctrl+C で終了コマンドが返らない")
	}
}

func TestLevelUpScreenShowsNewKana(t *testing.T) {
	// かなが増える昇格ではレベルアップ画面に新しいかなが出る
	m := newTestModel(t)
	m.leveledUp = true
	m.kanaAdded = true
	view := m.View()
	newest := m.drill.Newest()
	for _, want := range []string{"あたらしいかな", newest} {
		if !strings.Contains(view, want) {
			t.Errorf("かな追加の画面に %q がない:\n%s", want, view)
		}
	}
}

func TestViewCentersInTerminal(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = next.(Model)

	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) != 40 {
		t.Fatalf("画面の行数 = %d, want 40 (端末の高さいっぱいに配置)", len(lines))
	}
	// 上下に余白ができ、本文は先頭行より下から始まる
	if strings.TrimSpace(lines[0]) != "" {
		t.Errorf("最上段に本文がある (縦中央になっていない):\n%s", view)
	}
	if !strings.Contains(view, "使えるかな") {
		t.Errorf("本文が描画されていない:\n%s", view)
	}
	// 本文の行は左端から始まらない (横中央になっている)
	for _, line := range lines {
		if strings.Contains(line, "使えるかな") && !strings.HasPrefix(line, " ") {
			t.Errorf("本文が左端に張り付いている: %q", line)
		}
	}
}

func TestProgressBarShowsWindow(t *testing.T) {
	m := newTestModel(t)

	// まだ何も打っていない: 残り枠だけ
	view := m.View()
	if !strings.Contains(view, "░") {
		t.Errorf("残り枠が描かれていない:\n%s", view)
	}
	if strings.Contains(view, "█") {
		t.Errorf("打つ前から成功・失敗が描かれている:\n%s", view)
	}

	// 1語成功すると最低1マスの成功が見える
	m = typeWord(t, m)
	view = m.View()
	if !strings.Contains(view, "█") {
		t.Errorf("成功のマスが描かれていない:\n%s", view)
	}
	if !strings.Contains(view, "░") {
		t.Errorf("窓が埋まっていないのに残り枠が消えた:\n%s", view)
	}
}

func TestIntervalBetweenWords(t *testing.T) {
	engine := naginata.NewEngine(80 * time.Millisecond)
	d := drill.New(drill.DefaultConfig(), 1, nil)
	gen := lesson.NewGenerator(lesson.DefaultConfig(), rand.New(rand.NewSource(1)))
	m, err := New(engine, d, gen, "", 500*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	first := m.drill.Word()
	m = typeWord(t, m)

	// 打ち終わってもすぐには次の単語が出ない
	if !m.waiting {
		t.Fatal("インターバルに入っていない")
	}
	if !slices.Equal(m.drill.Word(), first) {
		t.Fatalf("インターバル中に単語が変わった: %v", m.drill.Word())
	}
	view := m.View()
	if !strings.Contains(view, "つぎの単語へ") {
		t.Errorf("インターバルの案内がない:\n%s", view)
	}

	// インターバル中のかなキーは無視され、ミスにならない
	m = pressChord(m, naginata.Set(naginata.KeyJ))
	if m.drill.WordErrors() != 0 {
		t.Fatalf("インターバル中の打鍵がミス扱いされた: %d", m.drill.WordErrors())
	}

	// インターバルが明けると次の単語が出て、計測が始まる
	next, _ := m.Update(tickMsg(time.Now().Add(time.Second)))
	m = next.(Model)
	if m.waiting {
		t.Fatal("インターバルが明けない")
	}
	if m.drill.Pos() != 0 || len(m.drill.Word()) == 0 {
		t.Fatalf("次の単語が始まっていない: word=%v pos=%d", m.drill.Word(), m.drill.Pos())
	}
	if e := m.drill.Elapsed(time.Now().Add(time.Second)); e > 2*time.Second {
		t.Errorf("計測がインターバル明けから始まっていない: %v", e)
	}
	// 明けたあとは普通に打てる
	m = typeWord(t, m)
	if _, total := m.drill.SuccessCount(); total != 2 {
		t.Errorf("インターバル後の単語が判定されていない")
	}
}
