// Package curriculum は練習するかなの拡張順序を定義する。
//
// 「ある」「ない」「する」を構成する5文字から始め、以後は
// ベースレイヤーの清音を頻度順に、次いでベース由来の濁音、基本の拗音、
// シフトレイヤーの清音、シフト由来の濁音、半濁音、濁音付き拗音、
// 外来音の順に1文字ずつ増やしていく。
package curriculum

import "slices"

// Group は同じ性質のかなのまとまりを表す。
type Group struct {
	Name  string
	Units []string
}

// Groups は拡張順に並んだグループの一覧。各グループ内は頻度順。
var Groups = []Group{
	{
		Name:  "はじめの5文字",
		Units: []string{"あ", "い", "な", "す", "る"},
	},
	{
		Name: "ベース清音",
		Units: []string{
			"ん", "し", "て", "か", "と", "た", "う", "く", "こ", "は",
			"き", "ら", "れ", "そ", "っ", "ろ", "け", "ほ", "ひ", "へ", "ー",
		},
	},
	{
		Name: "ベース濁音",
		Units: []string{
			"で", "が", "だ", "じ", "ど", "ず", "ば", "ぐ", "ご", "げ",
			"ぞ", "び", "ぼ", "べ", "ぎ",
		},
	},
	{
		Name: "基本拗音",
		Units: []string{
			"しょ", "しゅ", "しゃ", "きょ", "きゅ", "きゃ", "ちょ", "ちゃ", "ちゅ",
			"りょ", "りゅ", "りゃ", "ひょ", "ひゃ", "ひゅ", "にょ", "にゅ", "にゃ",
			"みょ", "みゃ", "みゅ",
		},
	},
	{
		Name: "シフト清音",
		Units: []string{
			"の", "に", "を", "も", "ま", "り", "お", "え", "さ", "つ", "せ",
			"よ", "わ", "ち", "や", "め", "み", "ゆ", "ね", "ふ", "む", "ぬ",
		},
	},
	{
		Name:  "シフト由来の濁音",
		Units: []string{"ぶ", "ざ", "ぜ", "づ", "ぢ"},
	},
	{
		Name:  "半濁音",
		Units: []string{"ぱ", "ぽ", "ぴ", "ぷ", "ぺ"},
	},
	{
		Name: "濁音・半濁音の拗音",
		Units: []string{
			"じょ", "じゅ", "じゃ", "ぎょ", "ぎゃ", "ぎゅ", "びょ", "びゃ", "びゅ",
			"ぴょ", "ぴゃ", "ぴゅ", "ぢゃ", "ぢゅ", "ぢょ",
		},
	},
	{
		Name: "外来音",
		Units: []string{
			"ゔ", "ふぁ", "ふぃ", "ふぇ", "ふぉ", "てぃ", "でぃ", "うぃ", "うぇ", "うぉ",
			"しぇ", "ちぇ", "じぇ", "とぅ", "どぅ", "てゅ", "でゅ", "ふゅ", "つぁ",
		},
	},
}

var flattened = func() []string {
	var all []string
	for _, g := range Groups {
		all = append(all, g.Units...)
	}
	return all
}()

// initialCount は最初のレベルで解放されているかなの数。
const initialCount = 5

// MaxLevel は最大レベルを返す。レベル1で5文字、以後1レベルごとに1文字増える。
func MaxLevel() int { return len(flattened) - initialCount + 1 }

// For はレベルに対応する解放済みかなの一覧を返す。
func For(level int) []string {
	if level < 1 {
		level = 1
	}
	if level > MaxLevel() {
		level = MaxLevel()
	}
	return flattened[:initialCount+level-1]
}

// GroupOf はかなが属するグループ名を返す。
func GroupOf(unit string) string {
	for _, g := range Groups {
		if slices.Contains(g.Units, unit) {
			return g.Name
		}
	}
	return ""
}
