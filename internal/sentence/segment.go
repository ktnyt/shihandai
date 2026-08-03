// Package sentence は練習用の例文を生成する。
// LFM2.5 (Ollama) による生成を試み、失敗した場合は単語バンクから組み立てる。
package sentence

import "strings"

// Segment は text を allowed の単位（拗音は2文字で1単位）に分割する。
// 最長一致で分割し、分割できない文字があれば ok=false を返す。
func Segment(text string, allowed []string) (units []string, ok bool) {
	set := make(map[string]bool, len(allowed))
	maxLen := 0
	for _, u := range allowed {
		set[u] = true
		if n := len([]rune(u)); n > maxLen {
			maxLen = n
		}
	}

	runes := []rune(text)
	for i := 0; i < len(runes); {
		matched := false
		for n := min(maxLen, len(runes)-i); n > 0; n-- {
			if set[string(runes[i:i+n])] {
				units = append(units, string(runes[i:i+n]))
				i += n
				matched = true
				break
			}
		}
		if !matched {
			return nil, false
		}
	}
	return units, true
}

// Normalize は生成結果を検証しやすい形に整える。
// 空白と句読点を除き、カタカナをひらがなに変換する。
func Normalize(text string) string {
	var b strings.Builder
	for _, r := range text {
		switch {
		case r == ' ' || r == '　' || r == '\n' || r == '\t' ||
			r == '、' || r == '。' || r == '，' || r == '．' ||
			r == '「' || r == '」' || r == '！' || r == '？' || r == '!' || r == '?':
			continue
		case r >= 'ァ' && r <= 'ヶ':
			b.WriteRune(r - 'ァ' + 'ぁ')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
