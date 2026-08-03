// Package store は練習の進捗をJSONで保存する。
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
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
	if errors.Is(err, fs.ErrNotExist) {
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
	// 手で編集されたファイルの null 値で落ちないようにする
	for unit, stat := range s.Stats {
		if stat == nil {
			delete(s.Stats, unit)
		}
	}
	return s, nil
}

// Encode は進捗をJSONに直列化する。
// Stats は並行に更新されうるので、非同期に書き込む場合は
// 先にこれを呼んでスナップショットを取ること。
func Encode(s State) ([]byte, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("進捗の変換に失敗: %w", err)
	}
	return data, nil
}

// Write は直列化済みの進捗を書き込む。
func Write(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("保存先の作成に失敗: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("一時ファイルの作成に失敗: %w", err)
	}
	defer os.Remove(tmp.Name()) // rename 済みなら失敗するだけ

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("進捗の書き込みに失敗: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("進捗の書き込みに失敗: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("進捗の保存に失敗: %w", err)
	}
	return nil
}

// Save は進捗を書き込む。
func Save(path string, s State) error {
	data, err := Encode(s)
	if err != nil {
		return err
	}
	return Write(path, data)
}
