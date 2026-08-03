package naginata

import (
	"testing"
	"time"
)

// press は複数キーを interval 間隔で順に押し、確定した文字を集める。
func press(t *testing.T, e *Engine, start time.Time, interval time.Duration, keys ...Key) ([]string, time.Time) {
	t.Helper()
	var out []string
	now := start
	for _, k := range keys {
		for _, em := range e.Press(k, now) {
			out = append(out, em.Text)
		}
		now = now.Add(interval)
	}
	return out, now
}

// flushAfterWindow はウィンドウ経過後の確定分を集める。
func flushAfterWindow(e *Engine, now time.Time) []string {
	var out []string
	for _, em := range e.Flush(now.Add(time.Second)) {
		out = append(out, em.Text)
	}
	return out
}

func TestEngine(t *testing.T) {
	const window = 80 * time.Millisecond
	start := time.Unix(0, 0)

	tests := []struct {
		name     string
		keys     []Key
		interval time.Duration
		want     []string // 押下中とウィンドウ経過後を合わせた確定列
	}{
		{"単打のあ", []Key{KeyJ}, 0, []string{"あ"}},
		{"単打のっ", []Key{KeyG}, 0, []string{"っ"}},
		{"連続入力あいなする", []Key{KeyJ, KeyK, KeyM, KeyO, KeyI}, 10 * time.Millisecond, []string{"あ", "い", "な", "す", "る"}},
		{"同時押しの濁音が", []Key{KeyF, KeyJ}, 10 * time.Millisecond, []string{"が"}},
		{"逆順の同時押しが", []Key{KeyJ, KeyF}, 10 * time.Millisecond, []string{"が"}},
		{"シフトのの", []Key{KeySpace, KeyJ}, 10 * time.Millisecond, []string{"の"}},
		{"後入れシフトのの", []Key{KeyJ, KeySpace}, 10 * time.Millisecond, []string{"の"}},
		{"拗音きゃ", []Key{KeyW, KeyH}, 10 * time.Millisecond, []string{"きゃ"}},
		{"濁音拗音ぎゃ", []Key{KeyJ, KeyW, KeyH}, 10 * time.Millisecond, []string{"ぎゃ"}},
		{"外来音ふぁ", []Key{KeyV, KeySemi, KeyJ}, 10 * time.Millisecond, []string{"ふぁ"}},
		{"半濁音ぱ", []Key{KeyM, KeyC}, 10 * time.Millisecond, []string{"ぱ"}},
		{"同じキーの連打", []Key{KeyJ, KeyJ}, 10 * time.Millisecond, []string{"あ", "あ"}},
		{"スペース単打", []Key{KeySpace}, 0, []string{" "}},
		{"かなのないキーは捨てる", []Key{KeyY}, 0, nil},
		{"バックスペース", []Key{KeyU}, 0, []string{"\b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEngine(window)
			got, now := press(t, e, start, tt.interval, tt.keys...)
			got = append(got, flushAfterWindow(e, now)...)
			if len(got) != len(tt.want) {
				t.Fatalf("確定列 = %q, want %q", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("確定列 = %q, want %q", got, tt.want)
				}
			}
		})
	}
}

func TestEngineWindowSeparatesChords(t *testing.T) {
	// ウィンドウを超えて押されたキーは同時押しにならない
	e := NewEngine(80 * time.Millisecond)
	start := time.Unix(0, 0)

	var got []string
	for _, em := range e.Press(KeyF, start) {
		got = append(got, em.Text)
	}
	// ウィンドウ経過で「か」が確定する
	for _, em := range e.Flush(start.Add(100 * time.Millisecond)) {
		got = append(got, em.Text)
	}
	// その後の J は単独の「あ」になる
	for _, em := range e.Press(KeyJ, start.Add(120*time.Millisecond)) {
		got = append(got, em.Text)
	}
	for _, em := range e.Flush(start.Add(300 * time.Millisecond)) {
		got = append(got, em.Text)
	}

	want := []string{"か", "あ"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("確定列 = %q, want %q", got, want)
	}
}

func TestEngineRolloverAcrossChord(t *testing.T) {
	// が(F+J) の直後に素早く J を押しても「が」「あ」になる
	e := NewEngine(80 * time.Millisecond)
	start := time.Unix(0, 0)

	got, now := press(t, e, start, 10*time.Millisecond, KeyF, KeyJ, KeyJ)
	got = append(got, flushAfterWindow(e, now)...)

	want := []string{"が", "あ"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("確定列 = %q, want %q", got, want)
	}
}

func TestEnginePressesCount(t *testing.T) {
	e := NewEngine(80 * time.Millisecond)
	start := time.Unix(0, 0)
	press(t, e, start, 10*time.Millisecond, KeyF, KeyJ, KeyK)
	if e.Presses() != 3 {
		t.Fatalf("Presses() = %d, want 3", e.Presses())
	}
}

func TestChordForAllTableEntries(t *testing.T) {
	// テーブルの全エントリが一意に確定できることを、単独入力で確かめる
	seen := map[string]bool{}
	for _, entry := range Table {
		if seen[entry.Text] {
			continue // 別名は正規の打鍵だけ確かめる
		}
		seen[entry.Text] = true

		e := NewEngine(80 * time.Millisecond)
		now := time.Unix(0, 0)
		var got []string
		for _, k := range entry.Keys.Keys() {
			for _, em := range e.Press(k, now) {
				got = append(got, em.Text)
			}
			now = now.Add(5 * time.Millisecond)
		}
		for _, em := range e.Flush(now.Add(time.Second)) {
			got = append(got, em.Text)
		}
		if len(got) != 1 || got[0] != entry.Text {
			t.Errorf("%v (%s): 確定列 = %q", entry.Keys.Label(), entry.Text, got)
		}
	}
}
