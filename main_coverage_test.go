package main

// 共通ヘルパーと main のカバレッジテスト

import (
	"os"
	"testing"
	"time"
)

// recoverCall は f を実行し、panic したら true を返す
func recoverCall(f func()) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	f()
	return false
}

// chdirTo はテスト中だけ作業ディレクトリを移動する
func chdirTo(t *testing.T, dir string) {
	t.Helper()
	saved, err := os.Getwd()
	if err != nil {
		t.Fatalf("カレントディレクトリ取得に失敗: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("ディレクトリ移動に失敗: %v", err)
	}
	t.Cleanup(func() { os.Chdir(saved) })
}

// TestWaitForConnectionCoverage 接続待機の成功とタイムアウトを検証する
func TestWaitForConnectionCoverage(t *testing.T) {
	dc := NewDroneController()

	if err := waitForConnection(dc, 3*time.Second); err != nil {
		t.Errorf("成功ケースでエラー: %v", err)
	}
	if err := waitForConnection(dc, 300*time.Millisecond); err == nil {
		t.Error("タイムアウトケースではエラーになるべき")
	}
}
