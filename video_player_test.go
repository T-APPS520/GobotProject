package main

import (
	"errors"
	"os/exec"
	"testing"

	"gobot.io/x/gobot/platforms/dji/tello"
)

// catSpec はテスト用のプレイヤー仕様（catは標準入力をそのまま読み捨てる）
func catSpec(t *testing.T) playerSpec {
	t.Helper()
	if _, err := exec.LookPath("cat"); err != nil {
		t.Skip("cat が無い環境ではスキップ")
	}
	return playerSpec{name: "cat"}
}

// TestVideoPlayerLifecycle プレイヤーの起動→書き込み→終了を確認
func TestVideoPlayerLifecycle(t *testing.T) {
	player := newVideoPlayer(catSpec(t))

	if player.Running() {
		t.Error("起動前は Running であってはならない")
	}

	if _, err := player.Write([]byte("frame")); err == nil {
		t.Error("起動前の Write はエラーになるべき")
	}

	if err := player.Start(); err != nil {
		t.Fatalf("プレイヤー起動に失敗: %v", err)
	}
	if !player.Running() {
		t.Error("起動後は Running であるべき")
	}

	// 二重起動してもエラーにならない
	if err := player.Start(); err != nil {
		t.Errorf("二重起動でエラー: %v", err)
	}

	n, err := player.Write([]byte{0x00, 0x00, 0x00, 0x01, 0x67})
	if err != nil {
		t.Errorf("フレーム書き込みに失敗: %v", err)
	}
	if n != 5 {
		t.Errorf("書き込みバイト数が不正: %d", n)
	}

	if err := player.Close(); err != nil {
		t.Errorf("プレイヤー終了に失敗: %v", err)
	}
	if player.Running() {
		t.Error("終了後は Running であってはならない")
	}

	// 二重終了してもエラーにならない
	if err := player.Close(); err != nil {
		t.Errorf("二重終了でエラー: %v", err)
	}
}

// TestVideoPlayerStartFailure 存在しないプレイヤーは起動エラーになる
func TestVideoPlayerStartFailure(t *testing.T) {
	player := newVideoPlayer(playerSpec{name: "gobot-nonexistent-player-xyz"})

	if err := player.Start(); err == nil {
		t.Error("存在しないプレイヤーの起動はエラーになるべき")
	}
	if player.Running() {
		t.Error("起動失敗後は Running であってはならない")
	}
}

// TestDetectPlayer 検出結果の整合性を確認
func TestDetectPlayer(t *testing.T) {
	spec, ok := detectPlayer()
	if !ok {
		t.Log("この環境に表示用プレイヤーは無い（その場合は false が正しい）")
		return
	}
	if spec.name == "" {
		t.Error("検出されたプレイヤー名が空")
	}
	if _, err := exec.LookPath(spec.name); err != nil {
		t.Errorf("検出されたプレイヤー %s が実行できない: %v", spec.name, err)
	}

	player, err := NewVideoPlayer()
	if err != nil {
		t.Fatalf("NewVideoPlayer に失敗: %v", err)
	}
	if player.Name() != spec.name {
		t.Errorf("プレイヤー名が検出結果と一致しない: %s != %s", player.Name(), spec.name)
	}
}

// TestCameraViewerDisplaysFrames ビューワーが受信フレームを表示へ回すことを確認
func TestCameraViewerDisplaysFrames(t *testing.T) {
	drone := tello.NewDriver("8890")
	cameraViewer := NewCameraViewer(drone)
	cameraViewer.playerFactory = func() (*VideoPlayer, error) {
		return newVideoPlayer(catSpec(t)), nil
	}

	cameraViewer.isRunning = true
	cameraViewer.processFrame([]byte{0x00, 0x00, 0x00, 0x01, 0x67, 0x42})
	cameraViewer.processFrame([]byte{0x00, 0x00, 0x00, 0x01, 0x68, 0x01})
	cameraViewer.processFrame([]byte{0x00, 0x00, 0x00, 0x01, 0x65, 0x88})

	if got := cameraViewer.GetDisplayedFrameCount(); got != 3 {
		t.Errorf("表示フレーム数が不正: %d", got)
	}
	if !cameraViewer.IsDisplaying() {
		t.Error("フレーム処理後は表示中であるべき")
	}

	cameraViewer.Stop()
	if cameraViewer.IsDisplaying() {
		t.Error("Stop 後は表示中であってはならない")
	}
}

// TestCameraViewerDisplayWithoutPlayer プレイヤーが無い環境でも受信処理が止まらない
func TestCameraViewerDisplayWithoutPlayer(t *testing.T) {
	drone := tello.NewDriver("8890")
	cameraViewer := NewCameraViewer(drone)
	cameraViewer.playerFactory = func() (*VideoPlayer, error) {
		return nil, errors.New("no player available")
	}

	cameraViewer.isRunning = true
	// パニックせず処理できること
	cameraViewer.processFrame([]byte{0x00, 0x00, 0x00, 0x01, 0x67})
	cameraViewer.processFrame([]byte{0x00, 0x00, 0x00, 0x01, 0x65})

	if got := cameraViewer.GetDisplayedFrameCount(); got != 0 {
		t.Errorf("プレイヤーが無いのに表示扱いになっている: %d", got)
	}
	if cameraViewer.IsDisplaying() {
		t.Error("プレイヤーが無いのに表示中になっている")
	}

	cameraViewer.Stop()
}
