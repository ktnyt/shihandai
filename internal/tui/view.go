package tui

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"

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

// View は画面を描画する。中身を組み立てて端末の中央に置く。
func (m Model) View() string {
	content := m.content()
	if m.width > 0 && m.height > 0 {
		// 幅を固定したブロックごと中央に置くと、行の長さが変わっても
		// 位置がぶれない
		block := lipgloss.NewStyle().Width(m.contentWidth()).Render(content)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, block)
	}
	return content
}

// contentWidth は画面中身の固定幅を返す。
func (m Model) contentWidth() int {
	return min(max(m.width-4, 40), 72)
}

func (m Model) content() string {
	if m.err != nil {
		return "エラー: " + m.err.Error() + "\n"
	}

	var b strings.Builder
	allowed := m.drill.Allowed()
	newest := allowed[len(allowed)-1]

	fmt.Fprintf(&b, "%s  %s\n",
		styleTitle.Render("shihandai — 薙刀式タイピング練習"),
		styleFaint.Render("(Esc で一時停止)"))
	fmt.Fprintf(&b, "レベル %d/%d  かな %d文字  ながさ %s  いまの段階: %s\n\n",
		m.drill.Level, curriculum.MaxLevel(), len(allowed),
		maxLenLabel(m.drill.Stage().MaxLen), curriculum.GroupOf(newest))

	fmt.Fprintf(&b, "%s %s\n\n",
		styleFaint.Render("使えるかな:"),
		wrapKana(allowed, max(m.contentWidth()-12, 28)))

	if m.leveledUp {
		fmt.Fprintf(&b, "  %s\n\n", styleNotice.Render(fmt.Sprintf("レベルアップ! レベル %d", m.drill.Level)))
		if m.kanaAdded {
			if chord, ok := naginata.ChordFor(newest); ok {
				fmt.Fprintf(&b, "  あたらしいかな: %s %s\n\n",
					styleCurrent.Render(newest),
					styleHint.Render("["+chord.Label()+"]"))
			}
		} else {
			fmt.Fprintf(&b, "  ながさ %s の語がでるようになった\n\n",
				maxLenLabel(m.drill.Stage().MaxLen))
		}
		b.WriteString("  " + styleFaint.Render("Space ではじめる、Esc で終了") + "\n")
		return b.String()
	}

	// 出題中の単語（入力済み、現在位置、残りで塗り分け）に続けて、
	// 先の単語を薄く並べる。打ち終わると右から左に流れてくる。
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
	for _, w := range m.upcoming {
		text := strings.Join(w, "")
		if m.paused {
			text = mask(text)
		}
		wordView.WriteString("  " + styleFaint.Render(text))
	}
	b.WriteString("  " + truncate.String(wordView.String(), uint(m.contentWidth()-2)) + "\n\n")

	// 次に打つかなの運指ヒント。一時停止・インターバル中は代わりに案内を出す
	switch {
	case m.paused:
		fmt.Fprintf(&b, "  %s %s\n",
			styleNotice.Render("一時停止中"),
			styleFaint.Render("(Space で同じ単語を最初から、Esc で終了)"))
	case m.waiting:
		b.WriteString("  " + styleFaint.Render("つぎの単語へ…") + "\n")
	default:
		if expected := m.drill.Expected(); expected != "" {
			if chord, ok := naginata.ChordFor(expected); ok {
				fmt.Fprintf(&b, "  つぎ: %s %s\n",
					styleCurrent.Render(expected),
					styleHint.Render("["+chord.Label()+"]"))
			}
		}
	}

	// 経過時間とミス
	var timer string
	if m.paused || m.waiting {
		timer = "--.-s"
	} else {
		timer = fmt.Sprintf("%5.1fs", m.drill.Elapsed(time.Now()).Seconds())
	}
	fmt.Fprintf(&b, "\n  %s   ミス %d\n", timer, m.drill.WordErrors())

	successes, total := m.drill.SuccessCount()
	b.WriteString("  " + m.progressBar(successes, total-successes) + "\n")
	b.WriteString("  " + m.kpmMeter() + "\n")
	kpm := m.drill.WindowKPM()
	missRate := m.drill.MissRate()
	fmt.Fprintf(&b, "  kpm %.0f/%.0f  ミス率 %.1f%%/%.1f%%  直近 %d/%d 語 (両方みたすとレベルアップ)\n",
		kpm, m.drill.Cfg.TargetKPM,
		missRate*100, m.drill.Cfg.MaxMissRate*100,
		total, m.drill.Cfg.WindowSize)
	fmt.Fprintf(&b, "  %s\n",
		styleFaint.Render(fmt.Sprintf("「%s」を含む語 %d/%d",
			newest, m.drill.NewKanaWords(), m.drill.GateTarget())))

	// 中央寄せしたときに縦位置がぶれないよう、空でも行を確保する
	flash := ""
	if m.flash != "" {
		flash = styleError.Render(m.flash)
	}
	b.WriteString("  " + flash + "\n")
	message := ""
	if m.message != "" {
		message = styleNotice.Render(m.message)
	}
	b.WriteString("  " + message + "\n")

	weak := ""
	if label := m.weakLabel(3); label != "" {
		weak = styleFaint.Render("にがて: " + label)
	}
	b.WriteString("\n" + weak + "\n")
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

// progressBar は成功率の窓を棒で描く。緑が成功、赤が失敗、
// 薄い部分が窓の残り（まだ打っていない分）。
func (m Model) progressBar(successes, failures int) string {
	window := m.drill.Cfg.WindowSize
	width := m.contentWidth() - 4
	if window <= 0 || width < 10 {
		return ""
	}

	sw := successes * width / window
	fw := failures * width / window
	// 1語でもあれば最低1マスは見えるようにする
	if successes > 0 && sw == 0 {
		sw = 1
	}
	if failures > 0 && fw == 0 {
		fw = 1
	}
	if over := sw + fw - width; over > 0 {
		sw -= over
	}
	rest := width - sw - fw

	return styleDone.Render(strings.Repeat("█", sw)) +
		styleError.Render(strings.Repeat("█", fw)) +
		styleFaint.Render(strings.Repeat("░", rest))
}

// kpmMeterMax はメーターの右端の値。
const kpmMeterMax = 180

// kpmMeter は直近の窓の kpm を 0〜180 のメーターで描く。
// 目標値の位置にマーカーを置き、達していれば緑、未達なら黄で塗る。
func (m Model) kpmMeter() string {
	width := m.contentWidth() - 4
	if width < 10 {
		return ""
	}
	kpm := m.drill.WindowKPM()
	fill := int(min(kpm, kpmMeterMax) / kpmMeterMax * float64(width))
	mark := int(m.drill.Cfg.TargetKPM / kpmMeterMax * float64(width))
	mark = min(max(mark, 0), width-1)

	fillStyle := styleHint
	if kpm >= m.drill.Cfg.TargetKPM {
		fillStyle = styleDone
	}
	var b strings.Builder
	for i := range width {
		switch {
		case i == mark:
			b.WriteString(styleNotice.Render("┃"))
		case i < fill:
			b.WriteString(fillStyle.Render("█"))
		default:
			b.WriteString(styleFaint.Render("░"))
		}
	}
	return b.String()
}

// maxLenLabel は長さ制限の表示名を返す。
func maxLenLabel(maxLen int) string {
	if maxLen <= 0 {
		return "せいげんなし"
	}
	return fmt.Sprintf("%d文字まで", maxLen)
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
