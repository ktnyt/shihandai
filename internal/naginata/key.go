// Package naginata は薙刀式v15配列の同時押し判定エンジンを提供する。
// かな定義は eswai/qmk_firmware users/naginata_v15 に準拠する。
package naginata

import "strings"

// Key は薙刀式で使う物理キー（QWERTY配列上の位置）を表す。
type Key uint8

const (
	KeyQ Key = iota
	KeyW
	KeyE
	KeyR
	KeyT
	KeyY
	KeyU
	KeyI
	KeyO
	KeyP
	KeyA
	KeyS
	KeyD
	KeyF
	KeyG
	KeyH
	KeyJ
	KeyK
	KeyL
	KeySemi
	KeyZ
	KeyX
	KeyC
	KeyV
	KeyB
	KeyN
	KeyM
	KeyComma
	KeyDot
	KeySlash
	KeySpace
	keyCount
)

var keyLabels = [keyCount]string{
	"Q", "W", "E", "R", "T", "Y", "U", "I", "O", "P",
	"A", "S", "D", "F", "G", "H", "J", "K", "L", ";",
	"Z", "X", "C", "V", "B", "N", "M", ",", ".", "/",
	"Space",
}

// Label はキーの表示名を返す。
func (k Key) Label() string {
	if k >= keyCount {
		return "?"
	}
	return keyLabels[k]
}

// KeyFromRune は端末から届いた文字を薙刀式のキーに変換する。
// 対応しない文字の場合は ok=false を返す。
func KeyFromRune(r rune) (Key, bool) {
	switch r {
	case 'q':
		return KeyQ, true
	case 'w':
		return KeyW, true
	case 'e':
		return KeyE, true
	case 'r':
		return KeyR, true
	case 't':
		return KeyT, true
	case 'y':
		return KeyY, true
	case 'u':
		return KeyU, true
	case 'i':
		return KeyI, true
	case 'o':
		return KeyO, true
	case 'p':
		return KeyP, true
	case 'a':
		return KeyA, true
	case 's':
		return KeyS, true
	case 'd':
		return KeyD, true
	case 'f':
		return KeyF, true
	case 'g':
		return KeyG, true
	case 'h':
		return KeyH, true
	case 'j':
		return KeyJ, true
	case 'k':
		return KeyK, true
	case 'l':
		return KeyL, true
	case ';':
		return KeySemi, true
	case 'z':
		return KeyZ, true
	case 'x':
		return KeyX, true
	case 'c':
		return KeyC, true
	case 'v':
		return KeyV, true
	case 'b':
		return KeyB, true
	case 'n':
		return KeyN, true
	case 'm':
		return KeyM, true
	case ',':
		return KeyComma, true
	case '.':
		return KeyDot, true
	case '/':
		return KeySlash, true
	case ' ':
		return KeySpace, true
	}
	return 0, false
}

// KeySet はキーの組み合わせをビットマスクで表す。
type KeySet uint32

// Set はキーの集合を作る。
func Set(keys ...Key) KeySet {
	var s KeySet
	for _, k := range keys {
		s |= 1 << k
	}
	return s
}

// Has は k が含まれるかを返す。
func (s KeySet) Has(k Key) bool { return s&(1<<k) != 0 }

// Count は含まれるキーの数を返す。
func (s KeySet) Count() int {
	c := 0
	for s > 0 {
		c += int(s & 1)
		s >>= 1
	}
	return c
}

// Keys は含まれるキーを列挙する。
func (s KeySet) Keys() []Key {
	var keys []Key
	for k := range keyCount {
		if s.Has(k) {
			keys = append(keys, k)
		}
	}
	return keys
}

// Label は "F+J" のような表示名を返す。Space は先頭に置く。
func (s KeySet) Label() string {
	var parts []string
	if s.Has(KeySpace) {
		parts = append(parts, KeySpace.Label())
	}
	for k := range keyCount {
		if k != KeySpace && s.Has(k) {
			parts = append(parts, k.Label())
		}
	}
	return strings.Join(parts, "+")
}
