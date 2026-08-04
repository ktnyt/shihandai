// Package dict は練習に使う日本語単語の読みを頻度順で提供する。
// データは SudachiDict から tools/mkdict で生成した words.txt を埋め込む。
package dict

import (
	_ "embed"
	"strings"
	"sync"
)

//go:embed words.txt
var raw string

var load = sync.OnceValue(func() []string {
	lines := strings.Split(raw, "\n")
	words := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		words = append(words, line)
	}
	return words
})

// Words はひらがな読みの単語を頻度の高い順で返す。
// 返り値を書き換えてはいけない。
func Words() []string { return load() }
