// Package store は練習の進捗をJSONで保存する。
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ktnyt/shihandai/internal/drill"
)

// State は保存される進捗。
type State struct {
	Level int                        `json:"level"`
	Stats map[string]*drill.UnitStat `json:"stats"`
}

// DefaultPath は保存先のパスを返す。
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("設定ディレクトリの取得に失敗: %w", err)
	}
	return filepath.Join(dir, "shihandai", "state.json"), nil
}

// Load は進捗を読み込む。ファイルがなければ初期状態を返す。
func Load(path string) (State, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return State{Level: 1, Stats: map[string]*drill.UnitStat{}}, nil
	}
	if err != nil {
		return State{}, fmt.Errorf("進捗の読み込みに失敗: %w", err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}, fmt.Errorf("進捗の解析に失敗: %w", err)
	}
	if s.Level < 1 {
		s.Level = 1
	}
	if s.Stats == nil {
		s.Stats = map[string]*drill.UnitStat{}
	}
	return s, nil
}

// Save は進捗を書き込む。
func Save(path string, s State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("保存先の作成に失敗: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("進捗の変換に失敗: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("進捗の書き込みに失敗: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("進捗の保存に失敗: %w", err)
	}
	return nil
}
