package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// TestVideoPlayerRestartSameInstance Close→Start で同じインスタンスを使い回しても
// 前のプロセスの終了監視が新しいプロセスの状態を壊さないことを確認
func TestVideoPlayerRestartSameInstance(t *testing.T) {
	player := newVideoPlayer(catSpec(t))

	if err := player.Start(); err != nil {
		t.Fatalf("1回目の起動に失敗: %v", err)
	}
	if err := player.Close(); err != nil {
		t.Fatalf("終了に失敗: %v", err)
	}

	// 同じインスタンスで再起動
	if err := player.Start(); err != nil {
		t.Fatalf("2回目の起動に失敗: %v", err)
	}

	// 前のプロセスの監視goroutineが発火しても、新しいプロセスは
	// 起動中のままであること（誤って新しいexitedチャネルを閉じないこと）
	time.Sleep(300 * time.Millisecond)
	if !player.Running() {
		t.Error("再起動したプレイヤーが起動中になっていない（前の監視が干渉している）")
	}

	// パニックせず終了できること（二重closeにならないこと）
	if err := player.Close(); err != nil {
		t.Errorf("再起動後の終了に失敗: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	if player.Running() {
		t.Error("終了後は Running であってはならない")
	}
}

// TestVideoPlayerCloseUnblocksWrite プレイヤーが入力を読まなくても
// Close でブロック中の書き込みが解き放たれることを確認（デッドロック防止）
func TestVideoPlayerCloseUnblocksWrite(t *testing.T) {
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep が無い環境ではスキップ")
	}
	// sleep は標準入力を一切読まないため、パイプが満杯になると Write が止まる
	player := newVideoPlayer(playerSpec{name: "sleep", args: []string{"30"}})
	if err := player.Start(); err != nil {
		t.Fatalf("プレイヤー起動に失敗: %v", err)
	}

	writeErr := make(chan error, 1)
	go func() {
		chunk := make([]byte, 64*1024)
		var err error
		for err == nil {
			_, err = player.Write(chunk)
		}
		writeErr <- err
	}()

	// パイプが満杯になり Write がブロックするまで待つ
	time.Sleep(500 * time.Millisecond)

	// Close が即座に戻り、ブロックしていた Write がエラーで終わること
	closeDone := make(chan struct{})
	go func() {
		player.Close()
		close(closeDone)
	}()
	select {
	case <-closeDone:
	case <-time.After(3 * time.Second):
		t.Fatal("Close がブロックしている（デッドロック）")
	}
	select {
	case err := <-writeErr:
		if err == nil {
			t.Error("Close 後の Write はエラーになるべき")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Close しても Write が解き放たれない（デッドロック）")
	}
}

// TestCameraViewerNoPlayerAfterStop Stop 後に処理途中のフレームが来ても
// プレイヤーを作り直さないことを確認
func TestCameraViewerNoPlayerAfterStop(t *testing.T) {
	drone := tello.NewDriver("8890")
	cameraViewer := NewCameraViewer(drone)
	cameraViewer.playerFactory = func() (*VideoPlayer, error) {
		return newVideoPlayer(catSpec(t)), nil
	}

	cameraViewer.isRunning = true
	cameraViewer.processFrame([]byte{0x00, 0x00, 0x00, 0x01, 0x67})
	if !cameraViewer.IsDisplaying() {
		t.Fatal("フレーム処理後は表示中であるべき")
	}
	displayed := cameraViewer.GetDisplayedFrameCount()

	cameraViewer.Stop()
	if cameraViewer.IsDisplaying() {
		t.Fatal("Stop 後は表示中であってはならない")
	}

	// Stop をすり抜けたフレームが直接 displayFrame に来ても起動しないこと
	cameraViewer.displayFrame([]byte{0x00, 0x00, 0x00, 0x01, 0x65})
	if cameraViewer.IsDisplaying() {
		t.Error("Stop 後にプレイヤーが作り直されている")
	}
	if got := cameraViewer.GetDisplayedFrameCount(); got != displayed {
		t.Errorf("Stop 後に表示フレーム数が増えている: %d != %d", got, displayed)
	}
}

// TestPlayerCandidateArgs プレイヤー候補の引数が各プレイヤーの仕様に合っているか確認
func TestPlayerCandidateArgs(t *testing.T) {
	specs := make(map[string]playerSpec)
	for _, c := range playerCandidates {
		specs[c.name] = c
	}

	// mpv のウィンドウタイトルは --title（--window-title は ffplay 用で mpv では起動エラーになる）
	mpv, ok := specs["mpv"]
	if !ok {
		t.Fatal("mpv の候補が無い")
	}
	foundTitle := false
	for _, arg := range mpv.args {
		if strings.HasPrefix(arg, "--window-title") {
			t.Errorf("mpv に未対応のオプションが含まれている: %s", arg)
		}
		if arg == "--title=Tello Camera" {
			foundTitle = true
		}
	}
	if !foundTitle {
		t.Error("mpv の引数に --title=Tello Camera が無い")
	}

	// ffplay は -window_title が正しい（こちらは変えないこと）
	ffplay, ok := specs["ffplay"]
	if !ok {
		t.Fatal("ffplay の候補が無い")
	}
	foundWindowTitle := false
	for i, arg := range ffplay.args {
		if arg == "-window_title" && i+1 < len(ffplay.args) && ffplay.args[i+1] == "Tello Camera" {
			foundWindowTitle = true
		}
	}
	if !foundWindowTitle {
		t.Error("ffplay の引数から -window_title Tello Camera が欠けている")
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

// TestDetectPlayerFoundCoverage プレイヤーが見つかる分岐（偽ffplayをPATHに置く）
func TestDetectPlayerFoundCoverage(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\nexec cat\n"
	ffplayPath := filepath.Join(dir, "ffplay")
	if err := os.WriteFile(ffplayPath, []byte(script), 0o755); err != nil {
		t.Fatalf("偽プレイヤーの作成に失敗: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	spec, ok := detectPlayer()
	if !ok {
		t.Fatal("偽ffplayが検出されなかった")
	}
	if spec.name != "ffplay" {
		t.Errorf("検出されたプレイヤーが不正: %s", spec.name)
	}

	player, err := NewVideoPlayer()
	if err != nil {
		t.Fatalf("NewVideoPlayer に失敗: %v", err)
	}
	if player.Name() != "ffplay" {
		t.Errorf("プレイヤー名が不正: %s", player.Name())
	}

	// 検出されたプレイヤーでのライフサイクル
	if err := player.Start(); err != nil {
		t.Fatalf("起動に失敗: %v", err)
	}
	if !player.Running() {
		t.Error("起動後は Running であるべき")
	}
	if _, err := player.Write([]byte{0x00, 0x00, 0x00, 0x01}); err != nil {
		t.Errorf("書き込みに失敗: %v", err)
	}
	if err := player.Close(); err != nil {
		t.Errorf("終了に失敗: %v", err)
	}
}

// TestVideoPlayerRunningStatesCoverage Running の各状態
func TestVideoPlayerRunningStatesCoverage(t *testing.T) {
	player := newVideoPlayer(catSpec(t))

	// 未起動（exited が nil）
	if player.Running() {
		t.Error("未起動では Running であってはならない")
	}

	if err := player.Start(); err != nil {
		t.Fatalf("起動に失敗: %v", err)
	}
	if !player.Running() {
		t.Error("起動後は Running であるべき")
	}

	if err := player.Close(); err != nil {
		t.Fatalf("終了に失敗: %v", err)
	}
	if player.Running() {
		t.Error("終了後は Running であってはならない")
	}
}

// TestDetectPlayerNotFoundCoverage プレイヤーが1つも無い環境の分岐
func TestDetectPlayerNotFoundCoverage(t *testing.T) {
	// PATH を空にして候補が見つからない状態にする
	t.Setenv("PATH", t.TempDir())

	if _, ok := detectPlayer(); ok {
		t.Error("プレイヤーが無いのに検出された")
	}
	if _, err := NewVideoPlayer(); err == nil {
		t.Error("プレイヤーが無いのに NewVideoPlayer が成功した")
	}
}

// TestVideoPlayerRunningAfterNaturalExitCoverage プロセスが自然終了した後の Running
func TestVideoPlayerRunningAfterNaturalExitCoverage(t *testing.T) {
	// "true" コマンドは即座に終了する
	player := newVideoPlayer(playerSpec{name: "true"})
	if err := player.Start(); err != nil {
		t.Fatalf("起動に失敗: %v", err)
	}

	
	waitForCondition(t, 2*time.Second, func() bool {
		return !player.Running()
	}, "自然終了後に Running が false にならない")

	// 終了後の Close も正常に終わる
	if err := player.Close(); err != nil {
		t.Errorf("自然終了後の Close でエラー: %v", err)
	}
}
