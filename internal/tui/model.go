// Package tui は bubbletea 製の練習画面を提供する。
package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ktnyt/shihandai/internal/drill"
	"github.com/ktnyt/shihandai/internal/naginata"
	"github.com/ktnyt/shihandai/internal/sentence"
	"github.com/ktnyt/shihandai/internal/store"
)

const tickInterval = 30 * time.Millisecond

type state int

const (
	stateLoading state = iota
	stateTyping
)

// Model は画面全体の状態。
type Model struct {
	engine *naginata.Engine
	drill  *drill.Drill
	gen    *sentence.Generator

	statePath string
	llmName   string

	state   state
	line    sentence.Line
	next    *lineMsg // 打鍵中に先読みした次の行
	pending bool     // 生成リクエストが飛んでいる間 true
	width   int
	message string
	flash   string
	err     error
}

// New は画面を作る。
func New(engine *naginata.Engine, d *drill.Drill, gen *sentence.Generator, statePath, llmName string) Model {
	return Model{
		engine:    engine,
		drill:     d,
		gen:       gen,
		statePath: statePath,
		llmName:   llmName,
		state:     stateLoading,
	}
}

type tickMsg time.Time

type lineMsg struct {
	level int // 生成時のレベル。今のレベルと違ったら捨てる
	line  sentence.Line
	err   error
}

func (m Model) tick() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// generate は現在のレベルに合わせて1行生成するコマンドを返す。
// すでにリクエストが飛んでいれば何もしない。
func (m *Model) generate() tea.Cmd {
	if m.pending {
		return nil
	}
	m.pending = true
	level := m.drill.Level
	allowed := m.drill.Allowed()
	gen := m.gen
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		line, err := gen.Generate(ctx, allowed)
		return lineMsg{level: level, line: line, err: err}
	}
}

// Init は最初の例文生成と定期処理を開始する。
func (m Model) Init() tea.Cmd {
	return tea.Batch((&m).generate(), m.tick())
}

// Update はイベントを処理する。
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.save()
			return m, tea.Quit
		case tea.KeyRunes:
			if m.state != stateTyping {
				return m, nil
			}
			now := time.Now()
			var cmds []tea.Cmd
			for _, r := range msg.Runes {
				if key, ok := naginata.KeyFromRune(r); ok {
					ems := m.engine.Press(key, now)
					m.drill.MarkKeydown(now, m.engine.Presses())
					if cmd := m.handleEmissions(ems, now); cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			}
			return m, tea.Batch(cmds...)
		case tea.KeySpace:
			if m.state != stateTyping {
				return m, nil
			}
			now := time.Now()
			ems := m.engine.Press(naginata.KeySpace, now)
			m.drill.MarkKeydown(now, m.engine.Presses())
			return m, m.handleEmissions(ems, now)
		}
		return m, nil

	case tickMsg:
		now := time.Time(msg)
		var cmd tea.Cmd
		if m.state == stateTyping {
			cmd = m.handleEmissions(m.engine.Flush(now), now)
		}
		return m, tea.Batch(m.tick(), cmd)

	case lineMsg:
		m.pending = false
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		if msg.level != m.drill.Level {
			// レベルが変わって古くなった行は捨てて作り直す
			if m.state == stateLoading {
				return m, m.generate()
			}
			return m, nil
		}
		if m.state == stateLoading {
			m.startLine(msg.line)
			// 打鍵中に次の行を先読みしておく
			return m, m.generate()
		}
		m.next = &msg
		return m, nil
	}
	return m, nil
}

func (m *Model) startLine(line sentence.Line) {
	m.line = line
	m.drill.StartLine(line.Units)
	m.state = stateTyping
	m.flash = ""
}

// handleEmissions は確定したかなを判定に流す。行が終わったら次の生成を返す。
func (m *Model) handleEmissions(ems []naginata.Emission, now time.Time) tea.Cmd {
	for _, em := range ems {
		switch m.drill.Input(em.Text) {
		case drill.ResultAdvance:
			m.flash = ""
		case drill.ResultError:
			m.flash = "ミス: " + printable(em.Text)
		case drill.ResultLineDone:
			out := m.drill.FinishLine(now, m.engine.Presses())
			m.message = outcomeMessage(out, m.drill.Cfg.TargetKPM)
			m.save()
			// 先読みした行がレベルと合っていればすぐ次を始める
			if m.next != nil && m.next.level == m.drill.Level {
				line := m.next.line
				m.next = nil
				m.startLine(line)
				return m.generate()
			}
			m.next = nil
			m.state = stateLoading
			return m.generate()
		}
	}
	return nil
}

func (m *Model) save() {
	if m.statePath == "" {
		return
	}
	_ = store.Save(m.statePath, store.State{Level: m.drill.Level, Stats: m.drill.Stats})
}

// Err は終了原因のエラーを返す。
func (m Model) Err() error { return m.err }
