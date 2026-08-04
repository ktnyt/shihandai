package tui

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/ktnyt/shihandai/internal/curriculum"
	"github.com/ktnyt/shihandai/internal/drill"
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
		styleFaint.Render("(Esc で終了)"))
	fmt.Fprintf(&b, "レベル %d/%d  かな %d文字  いまの段階: %s\n\n",
		m.drill.Level, curriculum.MaxLevel(), len(allowed), curriculum.GroupOf(newest))

	fmt.Fprintf(&b, "%s %s\n\n",
		styleFaint.Render("使えるかな:"),
		wrapKana(allowed, max(m.width-8, 40)))

	// 単語列（入力済み、現在位置、残りで塗り分け、単語の間は空ける）
	pos := m.drill.Pos()
	var lineView strings.Builder
	idx := 0
	for wi, word := range m.line.Words {
		if wi > 0 {
			lineView.WriteString(" ")
		}
		for _, u := range word {
			switch {
			case idx < pos:
				lineView.WriteString(styleDone.Render(u))
			case idx == pos:
				lineView.WriteString(styleCurrent.Render(u))
			default:
				lineView.WriteString(styleTodo.Render(u))
			}
			idx++
		}
	}
	b.WriteString("  " + lineView.String() + "\n\n")

	// 次に打つかなの運指ヒント
	if expected := m.drill.Expected(); expected != "" {
		if chord, ok := naginata.ChordFor(expected); ok {
			fmt.Fprintf(&b, "  つぎ: %s %s\n",
				styleCurrent.Render(expected),
				styleHint.Render("["+chord.Label()+"]"))
		}
	}

	// ステータス行
	kpm := m.drill.KPM(time.Now(), m.engine.Presses())
	fmt.Fprintf(&b, "\n  kpm %5.0f / %.0f   ミス %d\n",
		kpm, m.drill.Cfg.TargetKPM, m.drill.LineErrors())

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

func outcomeMessage(out drill.Outcome, targetKPM float64) string {
	switch {
	case out.Demoted:
		return fmt.Sprintf("「%s」の正答率が下がったのでレベルダウン (kpm %.0f)", out.WeakUnit, out.KPM)
	case out.Promoted:
		return fmt.Sprintf("ノーミス kpm %.0f! 新しいかなを追加", out.KPM)
	case out.Errors == 0:
		return fmt.Sprintf("ノーミス kpm %.0f (昇格には %.0f 必要)", out.KPM, targetKPM)
	default:
		return fmt.Sprintf("kpm %.0f  ミス %d", out.KPM, out.Errors)
	}
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
