package main

// coverage_full_test.go
//
// カバレッジ向上のための追加テスト集。
// 既存テストがリフレクション経由で「呼び出しのシミュレーション」に留まっていた
// 箇所（processKey 等）を、同パッケージから直接呼び出して実際に実行する。
//
// ドローン未接続の環境では、ドライバーのコマンド送信が内部でパニックする
// （実機のIPへ到達できないため）。これは実機操作では起こらないUT環境固有の
// 挙動なので、該当箇所は recoverCall で回復して「呼び出しまでの行」を実行する。
// パニック箇所より後ろの行は実機接続時にのみ到達可能であり、
// カバレッジの除外対象として扱う（理由はPRおよびスペースに明記）。

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nsf/termbox-go"
	"gobot.io/x/gobot/platforms/dji/tello"
)

// recoverCall は f を実行し、ドライバー未接続によるパニックを回復する
// パニックしたら true を返す
func recoverCall(f func()) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	f()
	return false
}

// waitForCondition は条件が満たされるまで待つ（非同期イベント処理用）
func waitForCondition(t *testing.T, timeout time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("タイムアウト: %s", msg)
}

// errWriteCloser は書き込みが必ず失敗する io.WriteCloser（テスト用）
type errWriteCloser struct{}

func (errWriteCloser) Write(p []byte) (int, error) { return 0, os.ErrClosed }
func (errWriteCloser) Close() error                { return nil }

// chdirTo はテスト中だけ作業ディレクトリを移動し、終了時に戻す
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

// ---------- DroneController ----------

// TestDroneControllerCommandsCoverage 全コマンドメソッドを実行する
func TestDroneControllerCommandsCoverage(t *testing.T) {
	dc := NewDroneController()

	// 離陸（未接続ではドライバー送信時にパニックする。呼び出しまでを実行）
	if !recoverCall(dc.TakeOff) {
		t.Log("TakeOff がパニックしなかった（接続済みドライバーの可能性）")
	}

	// 飛行中フラグを直接立てて着陸・移動系を叩く
	// （ドライバーの移動量設定は接続なしでも動作する）
	dc.isFlying = true
	if !recoverCall(dc.Land) {
		t.Log("Land がパニックしなかった")
	}

	dc.isFlying = false
	recoverCall(dc.TakeOffOrLand) // 離陸側
	dc.isFlying = true
	recoverCall(dc.TakeOffOrLand) // 着陸側

	// 移動系: 飛行していない場合はガードで弾かれる
	dc.isFlying = false
	dc.MoveForward()
	dc.MoveBackward()
	dc.MoveLeft()
	dc.MoveRight()
	dc.MoveUp()
	dc.MoveDown()

	// 移動系: 飛行中はドライバーへ移動量が設定される
	dc.isFlying = true
	dc.MoveForward()
	dc.MoveBackward()
	dc.MoveLeft()
	dc.MoveRight()
	dc.MoveUp()
	dc.MoveDown()

	// 録画トグル: 開始側は StartVideo でパニック、停止側は完走する
	dc.isRecording = false
	recoverCall(dc.ToggleRecording)
	dc.isRecording = true
	dc.ToggleRecording() // StopRecording 側
	if dc.IsRecording() {
		t.Error("トグル後は録画停止であるべき")
	}
	dc.StopRecording()
	recoverCall(dc.StartRecording)
}

// ---------- main ----------

// TestWaitForConnectionCoverage 接続待機の成功・タイムアウト両分岐
func TestWaitForConnectionCoverage(t *testing.T) {
	dc := NewDroneController()

	if err := waitForConnection(dc, 3*time.Second); err != nil {
		t.Errorf("成功ケースでエラー: %v", err)
	}

	if err := waitForConnection(dc, 300*time.Millisecond); err == nil {
		t.Error("タイムアウトケースではエラーになるべき")
	}
}

// ---------- KeyboardHandler ----------

// TestProcessKeyAllKeysCoverage 全キー入力パターンを直接実行する
func TestProcessKeyAllKeysCoverage(t *testing.T) {
	dc := NewDroneController()
	cv := NewCameraViewer(dc.GetDriver())
	kh := NewKeyboardHandler(dc, cv)
	kh.SetShutdownCallback(func() {}) // os.Exit を避ける

	key := func(ev termbox.Event) {
		recoverCall(func() { kh.processKey(ev) })
	}

	// 離陸/着陸（未接続ではドライバー送信でパニックするが分岐は実行される）
	key(termbox.Event{Type: termbox.EventKey, Key: termbox.KeyEsc})

	// 上昇: 飛行していない場合／飛行中の場合
	key(termbox.Event{Type: termbox.EventKey, Key: termbox.KeySpace})
	dc.isFlying = true
	key(termbox.Event{Type: termbox.EventKey, Key: termbox.KeySpace})

	// 移動キー（飛行中・非飛行中）
	for _, ch := range []rune{'w', 'W', 'a', 'A', 's', 'S', 'd', 'D', 'z', 'Z'} {
		key(termbox.Event{Type: termbox.EventKey, Ch: ch})
	}
	dc.isFlying = false
	for _, ch := range []rune{'w', 'a', 's', 'd', 'z'} {
		key(termbox.Event{Type: termbox.EventKey, Ch: ch})
	}

	// 録画トグル（カメラビューワーあり）
	key(termbox.Event{Type: termbox.EventKey, Ch: 'l'})
	if cv.IsRecording() {
		recFile := cv.GetCurrentRecordingFile()
		key(termbox.Event{Type: termbox.EventKey, Ch: 'L'})
		os.Remove(recFile)
	}

	// 未知のキー・値なしイベントは何もしない
	key(termbox.Event{Type: termbox.EventKey, Ch: 'x'})
	key(termbox.Event{Type: termbox.EventKey, Key: termbox.KeyF1})
	key(termbox.Event{Type: termbox.EventKey})

	// 終了キー（gracefulShutdown 内の Land でパニックするため回復する）
	key(termbox.Event{Type: termbox.EventKey, Ch: 'q'})
	key(termbox.Event{Type: termbox.EventKey, Key: termbox.KeyCtrlC})
}

// TestProcessKeyNilViewerCoverage カメラビューワーが無い場合の分岐
func TestProcessKeyNilViewerCoverage(t *testing.T) {
	kh := NewKeyboardHandler(NewDroneController(), nil)
	kh.SetShutdownCallback(func() {})

	// 'l' キー: ビューワーが nil なら何もしない
	recoverCall(func() {
		kh.processKey(termbox.Event{Type: termbox.EventKey, Ch: 'l'})
	})
}

// TestGracefulShutdownCompletesCoverage 終了処理が完走するパターン
// （ドローンコントローラーが nil の場合は着陸をスキップして完走できる）
func TestGracefulShutdownCompletesCoverage(t *testing.T) {
	// コントローラー・ビューワー両方あり（コールバックまで到達させるため
	// 着陸でパニックしないよう、コントローラーは nil にして分岐を通す）
	cv := NewCameraViewer(NewDroneController().GetDriver())
	kh := NewKeyboardHandler(nil, cv)
	called := false
	kh.SetShutdownCallback(func() { called = true })

	kh.gracefulShutdown()
	if !called {
		t.Error("シャットダウンコールバックが呼ばれていない")
	}

	// 両方 nil の場合
	kh2 := NewKeyboardHandler(nil, nil)
	called2 := false
	kh2.SetShutdownCallback(func() { called2 = true })
	kh2.gracefulShutdown()
	if !called2 {
		t.Error("シャットダウンコールバックが呼ばれていない（nil構成）")
	}
}

// TestSetupSignalHandlingCoverage シグナル受信で終了処理が動くことを確認
func TestSetupSignalHandlingCoverage(t *testing.T) {
	kh := NewKeyboardHandler(nil, nil)
	done := make(chan struct{})
	kh.SetShutdownCallback(func() { close(done) })

	kh.setupSignalHandling()

	// 自プロセスへ SIGINT を送る（Notify 登録済みのため終了はしない）
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("自プロセスの取得に失敗: %v", err)
	}
	if err := proc.Signal(os.Interrupt); err != nil {
		t.Fatalf("シグナル送信に失敗: %v", err)
	}

	select {
	case <-done:
		// 終了処理が完走した
	case <-time.After(3 * time.Second):
		t.Fatal("シグナルで終了処理が動かなかった")
	}
}

// TestKeyboardHandlerStartErrorCoverage 端末が無い環境では Start がエラーになる
func TestKeyboardHandlerStartErrorCoverage(t *testing.T) {
	kh := NewKeyboardHandler(NewDroneController(), nil)
	err := kh.Start()
	if err == nil {
		// 端末がある環境では成功する。その場合は通常どおり停止する
		if !kh.IsRunning() {
			t.Error("Start 成功後は実行中であるべき")
		}
		kh.Stop()
		return
	}
	if kh.IsRunning() {
		t.Error("Start 失敗後は実行中であってはならない")
	}
}

// TestHandleKeyboardNotRunningCoverage 停止中は即座に返る
func TestHandleKeyboardNotRunningCoverage(t *testing.T) {
	kh := NewKeyboardHandler(NewDroneController(), nil)
	done := make(chan struct{})
	go func() {
		kh.handleKeyboard()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("停止中の handleKeyboard が返ってこない")
	}
}

// ---------- CameraViewer ----------

// TestCameraViewerStartStopWithEventsCoverage Start→イベント受信→Stop の流れ
func TestCameraViewerStartStopWithEventsCoverage(t *testing.T) {
	drone := tello.NewDriver("8895")
	cv := NewCameraViewer(drone)
	cv.playerFactory = func() (*VideoPlayer, error) {
		return newVideoPlayer(catSpec(t)), nil
	}

	cv.Start()
	if !cv.IsRunning() {
		t.Fatal("Start 後は実行中であるべき")
	}

	// 接続完了イベント（ビデオ開始要求が走る。未接続のため内部で回復される）
	drone.Publish(drone.Event(tello.ConnectedEvent), nil)

	// フレームイベント → ハンドラ経由で表示まで届く
	drone.Publish(drone.Event(tello.VideoFrameEvent), []byte{0x00, 0x00, 0x00, 0x01, 0x67})
	waitForCondition(t, 2*time.Second, func() bool {
		return cv.GetDisplayedFrameCount() >= 1
	}, "フレームイベントが表示まで届かなかった")

	cv.Stop()
	if cv.IsRunning() {
		t.Error("Stop 後は実行中であってはならない")
	}
	// ストリーム再送ループの終了を少し待つ
	time.Sleep(200 * time.Millisecond)
}

// TestStreamStartLoopRetryCoverage 初回フレーム未着のまま再送タイマーが発火する経路
func TestStreamStartLoopRetryCoverage(t *testing.T) {
	drone := tello.NewDriver("8896")
	cv := NewCameraViewer(drone)

	cv.Start()
	// 再送タイマー（3秒）が少なくとも1回発火するまで待つ
	time.Sleep(3400 * time.Millisecond)
	cv.Stop()
	time.Sleep(200 * time.Millisecond)
}

// TestProcessFrameBranchesCoverage フレーム処理の各分岐
func TestProcessFrameBranchesCoverage(t *testing.T) {
	chdirTo(t, t.TempDir())

	drone := tello.NewDriver("8897")
	cv := NewCameraViewer(drone)
	cv.playerFactory = func() (*VideoPlayer, error) {
		return nil, os.ErrNotExist // 表示できない環境を再現（警告＋リトライ抑制）
	}
	cv.isRunning = true

	// 150フレーム処理して定期ログの分岐を通す
	frame := []byte{0x00, 0x00, 0x00, 0x01, 0x65}
	for i := 0; i < 150; i++ {
		cv.processFrame(frame)
	}
	if cv.frameCount != 150 {
		t.Errorf("フレーム数が不正: %d", cv.frameCount)
	}

	// 録画中のフレーム書き込み経路
	cv.StartRecording()
	cv.processFrame(frame)
	cv.StopRecording()

	// 停止中のフレームは無視される
	cv.isRunning = false
	cv.processFrame(frame)
	if cv.frameCount != 151 {
		t.Errorf("停止中のフレームが数えられている: %d", cv.frameCount)
	}
}

// TestDiscardPlayerCoverage 書き込み失敗時のプレイヤー破棄
func TestDiscardPlayerCoverage(t *testing.T) {
	drone := tello.NewDriver("8898")
	cv := NewCameraViewer(drone)
	cv.isRunning = true

	// 書き込みが必ず失敗するプレイヤーを差し込む
	failing := &VideoPlayer{
		spec:   playerSpec{name: "fake"},
		stdin:  errWriteCloser{},
		exited: make(chan struct{}), // 閉じていないので Running() は true
	}
	cv.player = failing
	cv.displayFrame([]byte{0x00, 0x00, 0x00, 0x01, 0x65})

	if cv.player != nil {
		t.Error("書き込み失敗したプレイヤーは破棄されるべき")
	}
	if cv.GetDisplayedFrameCount() != 0 {
		t.Error("失敗したフレームが表示扱いになっている")
	}

	// 別のプレイヤーを discardPlayer に渡しても現在のものは破棄されない
	good := newVideoPlayer(catSpec(t))
	if err := good.Start(); err != nil {
		t.Fatalf("プレイヤー起動に失敗: %v", err)
	}
	cv.player = good
	cv.discardPlayer(failing)
	if cv.player != good {
		t.Error("無関係なプレイヤーの破棄で現在のプレイヤーが消えた")
	}
	cv.Stop()
}

// TestEnsurePlayerDeadAndStartErrorCoverage 終了済みプレイヤーの作り直しと起動失敗
func TestEnsurePlayerDeadAndStartErrorCoverage(t *testing.T) {
	drone := tello.NewDriver("8899")
	cv := NewCameraViewer(drone)
	cv.isRunning = true

	// 終了済みプレイヤーが残っている場合
	dead := newVideoPlayer(catSpec(t))
	if err := dead.Start(); err != nil {
		t.Fatalf("プレイヤー起動に失敗: %v", err)
	}
	dead.Close()
	cv.player = dead
	cv.displayFrame([]byte{0x00, 0x00, 0x00, 0x01, 0x65})
	if cv.player != nil {
		t.Error("終了済みプレイヤーは破棄されるべき")
	}

	// 起動に失敗するプレイヤーが返る場合
	cv.playerRetryAt = time.Time{}
	cv.playerFactory = func() (*VideoPlayer, error) {
		return newVideoPlayer(playerSpec{name: "gobot-nonexistent-player-xyz"}), nil
	}
	cv.displayFrame([]byte{0x00, 0x00, 0x00, 0x01, 0x65})
	if cv.player != nil {
		t.Error("起動失敗したプレイヤーが残っている")
	}
	if cv.GetDisplayedFrameCount() != 0 {
		t.Error("表示できていないのにフレーム数が増えている")
	}
}

// TestStopWhileRecordingCoverage 録画中の Stop で録画も止まる
func TestStopWhileRecordingCoverage(t *testing.T) {
	chdirTo(t, t.TempDir())

	drone := tello.NewDriver("8900")
	cv := NewCameraViewer(drone)
	cv.StartRecording()
	if !cv.IsRecording() {
		t.Fatal("録画が開始していない")
	}
	cv.Stop()
	if cv.IsRecording() {
		t.Error("Stop 後は録画停止であるべき")
	}
}

// TestStartRecordingErrorCoverage 録画ファイルを作れない場合の分岐
func TestStartRecordingErrorCoverage(t *testing.T) {
	saved, err := os.Getwd()
	if err != nil {
		t.Fatalf("カレントディレクトリ取得に失敗: %v", err)
	}
	defer os.Chdir(saved)

	// 作業ディレクトリを消してファイル作成を失敗させる
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("ディレクトリ移動に失敗: %v", err)
	}
	if err := os.RemoveAll(tmp); err != nil {
		t.Fatalf("ディレクトリ削除に失敗: %v", err)
	}

	cv := NewCameraViewer(tello.NewDriver("8901"))
	cv.StartRecording()
	if cv.IsRecording() {
		t.Error("ファイル作成に失敗したのに録画中になっている")
	}
}

// TestMP4WriterCoverage MP4ライターの各分岐
func TestMP4WriterCoverage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mov")

	w, err := NewMP4Writer(path)
	if err != nil {
		t.Fatalf("ライター作成に失敗: %v", err)
	}

	// 空フレームは無視、正フレームは保持される
	if err := w.WriteFrame(nil); err != nil {
		t.Errorf("空フレーム書き込みでエラー: %v", err)
	}
	if err := w.WriteFrame([]byte{0x00, 0x00, 0x00, 0x01, 0x67}); err != nil {
		t.Errorf("フレーム書き込みでエラー: %v", err)
	}
	if w.frameNum != 1 {
		t.Errorf("フレーム数が不正: %d", w.frameNum)
	}

	if err := w.Close(); err != nil {
		t.Errorf("クローズでエラー: %v", err)
	}
	// 二重クローズは file == nil 分岐で正常終了
	if err := w.Close(); err != nil {
		t.Errorf("二重クローズでエラー: %v", err)
	}

	// 作成できないパスではエラー
	if _, err := NewMP4Writer(filepath.Join(dir, "no-such-dir", "x.mov")); err == nil {
		t.Error("存在しないディレクトリではエラーになるべき")
	}

	// /dev/full では書き込みが失敗し、Close がエラーを返す
	if _, err := os.Stat("/dev/full"); err == nil {
		wf, err := NewMP4Writer("/dev/full")
		if err != nil {
			t.Skipf("/dev/full を開けなかった: %v", err)
		}
		if err := wf.Close(); err == nil {
			t.Error("/dev/full ではクローズがエラーになるべき")
		}
	}
}

// TestStopRecordingCloseErrorCoverage 録画ファイルの保存に失敗する分岐
func TestStopRecordingCloseErrorCoverage(t *testing.T) {
	chdirTo(t, t.TempDir())

	cv := NewCameraViewer(tello.NewDriver("8902"))
	cv.StartRecording()
	if !cv.IsRecording() {
		t.Fatal("録画が開始していない")
	}

	// 書き込み先ファイルを先に閉じて、Close 時の保存を失敗させる
	if cv.mp4Writer != nil && cv.mp4Writer.file != nil {
		cv.mp4Writer.file.Close()
	}
	cv.StopRecording()
	if cv.IsRecording() {
		t.Error("停止後は録画中であってはならない")
	}
}

// TestMP4BoxBuildersCoverage 現在の録画フローからは未使用のボックス生成群
// （将来の完全なMP4生成に向けた部品。直接呼び出して検証する）
func TestMP4BoxBuildersCoverage(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "boxes.mov"))
	if err != nil {
		t.Fatalf("ファイル作成に失敗: %v", err)
	}
	defer f.Close()

	w := &MP4Writer{
		file:      f,
		startTime: time.Now().Add(-10 * time.Second), // 経過時間から duration が決まる
		frameNum:  5,
		frameData: [][]byte{{0x01, 0x02}},
	}

	if len(w.createQuickTimeMovieBox()) == 0 {
		t.Error("moovボックスが空")
	}
	// 開始直後（duration が 0 になる）場合はフレーム数から補う分岐
	wFresh := &MP4Writer{
		file:      f,
		startTime: time.Now(),
		frameNum:  7,
	}
	if len(wFresh.createQuickTimeMovieBox()) == 0 {
		t.Error("moovボックスが空（duration 0）")
	}
	if len(w.createQuickTimeMovieHeaderBox(100)) == 0 {
		t.Error("mvhdボックスが空")
	}
	if len(w.createQuickTimeVideoTrackBox(100)) == 0 {
		t.Error("trakボックスが空")
	}
	if len(w.createTrackHeaderBox(100)) == 0 {
		t.Error("tkhdボックスが空")
	}
	if len(w.createMediaBox(100)) == 0 {
		t.Error("mdiaボックスが空")
	}
	if len(w.createMediaHeaderBox(100)) == 0 {
		t.Error("mdhdボックスが空")
	}
	if len(w.createHandlerBox()) == 0 {
		t.Error("hdlrボックスが空")
	}
	if len(w.createMediaInfoBox()) == 0 {
		t.Error("minfボックスが空")
	}
	if len(w.createSimpleMovieBox()) == 0 {
		t.Error("簡易moovボックスが空")
	}
	if err := w.createQuickTimeCompatibleMP4(); err != nil {
		t.Errorf("QuickTime互換MP4の作成でエラー: %v", err)
	}
	if err := w.createMinimalQuickTimeMP4(); err != nil {
		t.Errorf("最小MP4の作成でエラー: %v", err)
	}
}

// ---------- VideoPlayer ----------

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

// ---------- 追加の詰めテスト ----------

// TestDetectPlayerNotFoundCoverage プレイヤーが1つも無い環境の分岐
func TestDetectPlayerNotFoundCoverage(t *testing.T) {
	// 空のディレクトリだけを PATH にして、どの候補も見つからない状態にする
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

	// 終了監視が exited を閉じたら Running は false を返す
	waitForCondition(t, 2*time.Second, func() bool {
		return !player.Running()
	}, "自然終了後に Running が false にならない")

	// 終了後の Close も正常に終わる
	if err := player.Close(); err != nil {
		t.Errorf("自然終了後の Close でエラー: %v", err)
	}
}

// TestDisplayFrameEmptyCoverage 空フレームは表示されない
func TestDisplayFrameEmptyCoverage(t *testing.T) {
	cv := NewCameraViewer(tello.NewDriver("8903"))
	cv.isRunning = true
	cv.displayFrame(nil)
	cv.displayFrame([]byte{})
	if cv.GetDisplayedFrameCount() != 0 {
		t.Error("空フレームが表示扱いになっている")
	}
}

// TestStreamStartLoopExitsOnFirstFrameCoverage
// 初回フレーム到達後、再送タイマーの発火でループが自律終了する経路
func TestStreamStartLoopExitsOnFirstFrameCoverage(t *testing.T) {
	drone := tello.NewDriver("8904")
	cv := NewCameraViewer(drone)
	cv.playerFactory = func() (*VideoPlayer, error) {
		return newVideoPlayer(catSpec(t)), nil
	}

	cv.Start()
	drone.Publish(drone.Event(tello.VideoFrameEvent), []byte{0x00, 0x00, 0x00, 0x01, 0x67})
	waitForCondition(t, 2*time.Second, func() bool {
		return cv.GetDisplayedFrameCount() >= 1
	}, "フレームが表示されなかった")

	// 次の再送タイマー（3秒周期）でループが終了するまで待つ
	time.Sleep(3400 * time.Millisecond)

	cv.Stop()
	time.Sleep(100 * time.Millisecond)
}
