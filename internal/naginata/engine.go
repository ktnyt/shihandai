package naginata

import "time"

// Emission はエンジンが確定した1文字（または拗音などの1単位）を表す。
type Emission struct {
	Text  string
	Chord KeySet
}

// Engine はキー押下の列から薙刀式のかなを確定する。
//
// 端末入力ではキーリリースを検出できないため、QMKの「リリースで確定」の
// 代わりにタイミングウィンドウを使う。ウィンドウ内に押されたキーを
// 同時押しとみなし、QMKと同じ最長一致の貪欲法で確定する。
type Engine struct {
	window   time.Duration
	buf      []Key
	deadline time.Time
	armed    bool
	presses  int
}

// NewEngine はウィンドウ幅を指定してエンジンを作る。
func NewEngine(window time.Duration) *Engine {
	return &Engine{window: window}
}

// Presses は起動からの総打鍵数を返す。
func (e *Engine) Presses() int { return e.presses }

// Pending は未確定のキーがあるかを返す。
func (e *Engine) Pending() bool { return len(e.buf) > 0 }

// Press はキー押下を処理し、確定したかなを返す。
func (e *Engine) Press(k Key, now time.Time) []Emission {
	e.presses++

	var out []Emission
	// 同じキーの連打は同時押しになり得ないので、先にバッファを確定する。
	if e.comb(len(e.buf)).Has(k) {
		out = append(out, e.flushAll()...)
	}
	e.buf = append(e.buf, k)

	for len(e.buf) > 0 {
		nc, complete := e.candidates()
		if nc == 0 {
			// もう組み合わせが伸びないので、先頭から確定できる分を確定する。
			out = append(out, e.typeOnce()...)
			continue
		}
		if nc == 1 && complete {
			out = append(out, e.flushAll()...)
		}
		break
	}

	if len(e.buf) > 0 {
		e.deadline = now.Add(e.window)
		e.armed = true
	} else {
		e.armed = false
	}
	return out
}

// Flush はウィンドウ経過後に未確定のキーを確定する。定期的に呼ぶ。
func (e *Engine) Flush(now time.Time) []Emission {
	if !e.armed || now.Before(e.deadline) {
		return nil
	}
	e.armed = false
	return e.flushAll()
}

// comb はバッファ先頭 n キーの組み合わせを返す。
func (e *Engine) comb(n int) KeySet {
	var s KeySet
	for _, k := range e.buf[:n] {
		s |= 1 << k
	}
	return s
}

// candidates はバッファ全体を含む組み合わせの候補数を返す。
// complete は候補が一つでかつ全キーが押されていることを示す。
func (e *Engine) candidates() (count int, complete bool) {
	comb := e.comb(len(e.buf))
	var hit KeySet
	for _, entry := range Table {
		if entry.Keys&comb == comb {
			count++
			hit = entry.Keys
		}
	}
	complete = count == 1 && len(e.buf) >= hit.Count()
	return count, complete
}

// typeOnce はバッファ先頭からの最長一致で1単位を確定する。
// どの長さでも一致しなければ先頭の1キーを捨てる。
func (e *Engine) typeOnce() []Emission {
	for nt := len(e.buf); nt > 0; nt-- {
		comb := e.comb(nt)
		for _, entry := range Table {
			if entry.Keys == comb {
				e.buf = e.buf[nt:]
				return []Emission{{Text: entry.Text, Chord: comb}}
			}
		}
	}
	e.buf = e.buf[1:]
	return nil
}

// flushAll はバッファが空になるまで確定を繰り返す。
func (e *Engine) flushAll() []Emission {
	var out []Emission
	for len(e.buf) > 0 {
		out = append(out, e.typeOnce()...)
	}
	return out
}
