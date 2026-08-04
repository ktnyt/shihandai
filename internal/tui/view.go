package tui

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/ktnyt/shihandai/internal/curriculum"
	"github.com/ktnyt/shihandai/internal/naginata"
)

var (
	styleTitle   = lipgloss.NewStyle().Bold(true)
	styleFaint   = lipgloss.NewStyle().Faint(true)
	styleDone    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "28", Dark: "42"})
	styleCurrent = lipgloss.NewStyle().Reverse(true).Bold(true)
	styleTodo    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "240", Dark: "250"})
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "160", Dark: "203"}).Bold(true)
	styleNotice  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "26", Dark: "75"}).Bold(true)
	styleHint    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "94", Dark: "220"})
)

// View は画面を描画する。
func (m Model) View() string {
	if m.err != nil {
		return "エラー: " + m.err.Error() + "\n"
	}

	var b strings.Builder
	allowed := m.drill.Allowed()
	newest := allowed[len(allowed)-1]

	fmt.Fprintf(&b, "%s  %s\n",
		styleTitle.Render("shihandai — 薙刀式タイピング練習"),
		styleFaint.Render("(Esc で一時停止)"))
	fmt.Fprintf(&b, "レベル %d/%d  かな %d文字  いまの段階: %s\n\n",
		m.drill.Level, curriculum.MaxLevel(), len(allowed), curriculum.GroupOf(newest))

	fmt.Fprintf(&b, "%s %s\n\n",
		styleFaint.Render("使えるかな:"),
		wrapKana(allowed, max(m.width-8, 40)))

	// 出題中の単語（入力済み、現在位置、残りで塗り分け）。
	// 一時停止中はレイアウトを保ったまま伏せ字にする
	pos := m.drill.Pos()
	var wordView strings.Builder
	for i, u := range m.drill.Word() {
		if m.paused {
			wordView.WriteString(styleFaint.Render(mask(u)))
			continue
		}
		switch {
		case i < pos:
			wordView.WriteString(styleDone.Render(u))
		case i == pos:
			wordView.WriteString(styleCurrent.Render(u))
		default:
			wordView.WriteString(styleTodo.Render(u))
		}
	}
	b.WriteString("  " + wordView.String() + "\n\n")

	// 次に打つかなの運指ヒント。一時停止中は代わりに案内を出す
	switch {
	case m.paused:
		fmt.Fprintf(&b, "  %s %s\n",
			styleNotice.Render("一時停止中"),
			styleFaint.Render("(Space で同じ単語を最初から、Esc で終了)"))
	default:
		if expected := m.drill.Expected(); expected != "" {
			if chord, ok := naginata.ChordFor(expected); ok {
				fmt.Fprintf(&b, "  つぎ: %s %s\n",
					styleCurrent.Render(expected),
					styleHint.Render("["+chord.Label()+"]"))
			}
		}
	}

	// 時間と成功率
	threshold := m.drill.Threshold()
	var timer string
	if m.paused {
		timer = fmt.Sprintf("--.-s / %.1fs", threshold.Seconds())
	} else {
		elapsed := m.drill.Elapsed(time.Now())
		timer = fmt.Sprintf("%5.1fs / %.1fs", elapsed.Seconds(), threshold.Seconds())
		if elapsed > threshold {
			timer = styleError.Render(timer)
		}
	}
	fmt.Fprintf(&b, "\n  %s   ミス %d\n", timer, m.drill.WordErrors())

	successes, total := m.drill.SuccessCount()
	rate := "-"
	if total > 0 {
		rate = fmt.Sprintf("%.0f%%", float64(successes)/float64(total)*100)
	}
	fmt.Fprintf(&b, "  直近 %d/%d 語 成功率 %s  (%d 語で %.0f%% を超えたらレベルアップ)\n",
		successes, total, rate,
		m.drill.Cfg.WindowSize, m.drill.Cfg.PromoteRate*100)

	if m.flash != "" {
		b.WriteString("  " + styleError.Render(m.flash) + "\n")
	}
	if m.message != "" {
		b.WriteString("  " + styleNotice.Render(m.message) + "\n")
	}

	if weak := m.weakLabel(3); weak != "" {
		b.WriteString("\n" + styleFaint.Render("にがて: "+weak) + "\n")
	}
	return b.String()
}

type weakItem struct {
	unit string
	acc  float64
}

// weakItems は直近正答率の低いかなを上位 n 件返す。
func (m Model) weakItems(n int) []weakItem {
	var items []weakItem
	for _, u := range m.drill.Allowed() {
		s, ok := m.drill.Stats[u]
		if !ok || s == nil || len(s.Recent) == 0 {
			continue
		}
		if acc := s.RecentAccuracy(); acc < 1 {
			items = append(items, weakItem{u, acc})
		}
	}
	// 同率のときに表示がちらつかないよう、かなを第2キーにして安定に並べる
	slices.SortStableFunc(items, func(a, b weakItem) int {
		if c := cmp.Compare(a.acc, b.acc); c != 0 {
			return c
		}
		return cmp.Compare(a.unit, b.unit)
	})
	if len(items) > n {
		items = items[:n]
	}
	return items
}

func (m Model) weakLabel(n int) string {
	var parts []string
	for _, it := range m.weakItems(n) {
		parts = append(parts, fmt.Sprintf("%s %.0f%%", it.unit, it.acc*100))
	}
	return strings.Join(parts, "  ")
}

func wrapKana(units []string, width int) string {
	joined := strings.Join(units, " ")
	if lipgloss.Width(joined) <= width {
		return joined
	}
	// 長くなったら末尾側を優先して表示する
	for i := range units {
		rest := strings.Join(units[i:], " ")
		if lipgloss.Width(rest)+2 <= width {
			return "… " + rest
		}
	}
	return joined
}

// mask はかなを同じ表示幅の伏せ字にする。
func mask(unit string) string {
	var b strings.Builder
	for range []rune(unit) {
		b.WriteRune('●')
	}
	return b.String()
}

func printable(text string) string {
	switch text {
	case " ":
		return "␣"
	case "\b":
		return "BS"
	case "\n":
		return "⏎"
	}
	return text
}
