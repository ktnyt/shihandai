package naginata

// Entry は同時押しの組み合わせと出力文字の対応を表す。
type Entry struct {
	Keys KeySet
	Text string
}

// 制御用の出力文字。
const (
	TextBackspace = "\b"
	TextEnter     = "\n"
)

// Table は薙刀式v15のかな変換テーブル。
// eswai/qmk_firmware users/naginata_v15/naginata_v15.c の ngmap を移植した。
// 編集モードとエンター付き句読点の入力は練習対象外なので含めない。
var Table = []Entry{
	// 清音
	{Set(KeyJ), "あ"},
	{Set(KeyK), "い"},
	{Set(KeyL), "う"},
	{Set(KeySpace, KeyO), "え"},
	{Set(KeySpace, KeyN), "お"},
	{Set(KeyF), "か"},
	{Set(KeyW), "き"},
	{Set(KeyH), "く"},
	{Set(KeyS), "け"},
	{Set(KeyV), "こ"},
	{Set(KeySpace, KeyU), "さ"},
	{Set(KeyR), "し"},
	{Set(KeyO), "す"},
	{Set(KeySpace, KeyA), "せ"},
	{Set(KeyB), "そ"},
	{Set(KeyN), "た"},
	{Set(KeySpace, KeyG), "ち"},
	{Set(KeySpace, KeyL), "つ"},
	{Set(KeyE), "て"},
	{Set(KeyD), "と"},
	{Set(KeyM), "な"},
	{Set(KeySpace, KeyD), "に"},
	{Set(KeySpace, KeyW), "ぬ"},
	{Set(KeySpace, KeyR), "ね"},
	{Set(KeySpace, KeyJ), "の"},
	{Set(KeyC), "は"},
	{Set(KeyX), "ひ"},
	{Set(KeySpace, KeyX), "ひ"},
	{Set(KeySpace, KeySemi), "ふ"},
	{Set(KeyP), "へ"},
	{Set(KeyZ), "ほ"},
	{Set(KeySpace, KeyZ), "ほ"},
	{Set(KeySpace, KeyF), "ま"},
	{Set(KeySpace, KeyB), "み"},
	{Set(KeySpace, KeyComma), "む"},
	{Set(KeySpace, KeyS), "め"},
	{Set(KeySpace, KeyK), "も"},
	{Set(KeySpace, KeyH), "や"},
	{Set(KeySpace, KeyP), "ゆ"},
	{Set(KeySpace, KeyI), "よ"},
	{Set(KeyDot), "ら"},
	{Set(KeySpace, KeyE), "り"},
	{Set(KeyI), "る"},
	{Set(KeySlash), "れ"},
	{Set(KeySpace, KeySlash), "れ"},
	{Set(KeyA), "ろ"},
	{Set(KeySpace, KeyDot), "わ"},
	{Set(KeySpace, KeyC), "を"},
	{Set(KeyComma), "ん"},
	{Set(KeySemi), "ー"},

	// 濁音
	{Set(KeyJ, KeyF), "が"},
	{Set(KeyJ, KeyW), "ぎ"},
	{Set(KeyF, KeyH), "ぐ"},
	{Set(KeyJ, KeyS), "げ"},
	{Set(KeyJ, KeyV), "ご"},
	{Set(KeyF, KeyU), "ざ"},
	{Set(KeyJ, KeyR), "じ"},
	{Set(KeyF, KeyO), "ず"},
	{Set(KeyJ, KeyA), "ぜ"},
	{Set(KeyJ, KeyB), "ぞ"},
	{Set(KeyF, KeyN), "だ"},
	{Set(KeyJ, KeyG), "ぢ"},
	{Set(KeyF, KeyL), "づ"},
	{Set(KeyJ, KeyE), "で"},
	{Set(KeyJ, KeyD), "ど"},
	{Set(KeyJ, KeyC), "ば"},
	{Set(KeyJ, KeyX), "び"},
	{Set(KeyF, KeySemi), "ぶ"},
	{Set(KeyF, KeyP), "べ"},
	{Set(KeyJ, KeyZ), "ぼ"},
	{Set(KeyQ), "ゔ"},

	// 半濁音
	{Set(KeyM, KeyC), "ぱ"},
	{Set(KeyM, KeyX), "ぴ"},
	{Set(KeyV, KeySemi), "ぷ"},
	{Set(KeyV, KeyP), "ぺ"},
	{Set(KeyM, KeyZ), "ぽ"},

	// 小書き
	{Set(KeyQ, KeyH), "ゃ"},
	{Set(KeyQ, KeyP), "ゅ"},
	{Set(KeyQ, KeyI), "ょ"},
	{Set(KeyQ, KeyJ), "ぁ"},
	{Set(KeyQ, KeyK), "ぃ"},
	{Set(KeyQ, KeyL), "ぅ"},
	{Set(KeyQ, KeyO), "ぇ"},
	{Set(KeyQ, KeyN), "ぉ"},
	{Set(KeyQ, KeyDot), "ゎ"},
	{Set(KeyG), "っ"},
	{Set(KeyQ, KeyS), "ヶ"},
	{Set(KeyQ, KeyF), "ヵ"},

	// 清音拗音 濁音拗音 半濁拗音
	{Set(KeyR, KeyH), "しゃ"},
	{Set(KeyR, KeyP), "しゅ"},
	{Set(KeyR, KeyI), "しょ"},
	{Set(KeyJ, KeyR, KeyH), "じゃ"},
	{Set(KeyJ, KeyR, KeyP), "じゅ"},
	{Set(KeyJ, KeyR, KeyI), "じょ"},
	{Set(KeyW, KeyH), "きゃ"},
	{Set(KeyW, KeyP), "きゅ"},
	{Set(KeyW, KeyI), "きょ"},
	{Set(KeyJ, KeyW, KeyH), "ぎゃ"},
	{Set(KeyJ, KeyW, KeyP), "ぎゅ"},
	{Set(KeyJ, KeyW, KeyI), "ぎょ"},
	{Set(KeyG, KeyH), "ちゃ"},
	{Set(KeyG, KeyP), "ちゅ"},
	{Set(KeyG, KeyI), "ちょ"},
	{Set(KeyJ, KeyG, KeyH), "ぢゃ"},
	{Set(KeyJ, KeyG, KeyP), "ぢゅ"},
	{Set(KeyJ, KeyG, KeyI), "ぢょ"},
	{Set(KeyD, KeyH), "にゃ"},
	{Set(KeyD, KeyP), "にゅ"},
	{Set(KeyD, KeyI), "にょ"},
	{Set(KeyX, KeyH), "ひゃ"},
	{Set(KeyX, KeyP), "ひゅ"},
	{Set(KeyX, KeyI), "ひょ"},
	{Set(KeyJ, KeyX, KeyH), "びゃ"},
	{Set(KeyJ, KeyX, KeyP), "びゅ"},
	{Set(KeyJ, KeyX, KeyI), "びょ"},
	{Set(KeyM, KeyX, KeyH), "ぴゃ"},
	{Set(KeyM, KeyX, KeyP), "ぴゅ"},
	{Set(KeyM, KeyX, KeyI), "ぴょ"},
	{Set(KeyB, KeyH), "みゃ"},
	{Set(KeyB, KeyP), "みゅ"},
	{Set(KeyB, KeyI), "みょ"},
	{Set(KeyE, KeyH), "りゃ"},
	{Set(KeyE, KeyP), "りゅ"},
	{Set(KeyE, KeyI), "りょ"},

	// 清音外来音 濁音外来音
	{Set(KeyM, KeyE, KeyK), "てぃ"},
	{Set(KeyM, KeyE, KeyP), "てゅ"},
	{Set(KeyJ, KeyE, KeyK), "でぃ"},
	{Set(KeyJ, KeyE, KeyP), "でゅ"},
	{Set(KeyM, KeyD, KeyL), "とぅ"},
	{Set(KeyJ, KeyD, KeyL), "どぅ"},
	{Set(KeyM, KeyR, KeyO), "しぇ"},
	{Set(KeyM, KeyG, KeyO), "ちぇ"},
	{Set(KeyJ, KeyR, KeyO), "じぇ"},
	{Set(KeyJ, KeyG, KeyO), "ぢぇ"},
	{Set(KeyV, KeySemi, KeyJ), "ふぁ"},
	{Set(KeyV, KeySemi, KeyK), "ふぃ"},
	{Set(KeyV, KeySemi, KeyO), "ふぇ"},
	{Set(KeyV, KeySemi, KeyN), "ふぉ"},
	{Set(KeyV, KeySemi, KeyP), "ふゅ"},
	{Set(KeyV, KeyK, KeyO), "いぇ"},
	{Set(KeyV, KeyL, KeyK), "うぃ"},
	{Set(KeyV, KeyL, KeyO), "うぇ"},
	{Set(KeyV, KeyL, KeyN), "うぉ"},
	{Set(KeyM, KeyQ, KeyJ), "ゔぁ"},
	{Set(KeyM, KeyQ, KeyK), "ゔぃ"},
	{Set(KeyM, KeyQ, KeyO), "ゔぇ"},
	{Set(KeyM, KeyQ, KeyN), "ゔぉ"},
	{Set(KeyM, KeyQ, KeyP), "ゔゅ"},
	{Set(KeyV, KeyH, KeyJ), "くぁ"},
	{Set(KeyV, KeyH, KeyK), "くぃ"},
	{Set(KeyV, KeyH, KeyO), "くぇ"},
	{Set(KeyV, KeyH, KeyN), "くぉ"},
	{Set(KeyV, KeyH, KeyDot), "くゎ"},
	{Set(KeyF, KeyH, KeyJ), "ぐぁ"},
	{Set(KeyF, KeyH, KeyK), "ぐぃ"},
	{Set(KeyF, KeyH, KeyO), "ぐぇ"},
	{Set(KeyF, KeyH, KeyN), "ぐぉ"},
	{Set(KeyF, KeyH, KeyDot), "ぐゎ"},
	{Set(KeyV, KeyL, KeyJ), "つぁ"},

	// 記号と制御
	{Set(KeySpace), " "},
	{Set(KeySpace, KeyV), "、"},
	{Set(KeySpace, KeyM), "。"},
	{Set(KeyU), TextBackspace},
	{Set(KeyV, KeyM), TextEnter},
}

// ChordFor は text を入力する組み合わせを返す。
// 別名がある場合は最初に定義されたもの（正規の打鍵）を返す。
func ChordFor(text string) (KeySet, bool) {
	for _, e := range Table {
		if e.Text == text {
			return e.Keys, true
		}
	}
	return 0, false
}
