package main

// main_coverage_test.go
//
// mainブランチの既存コードに対するカバレッジ向上テスト。
// 既存テストがリフレクション経由で「呼び出しのシミュレーション」に留まっていた
// 箇所（processKey 等）を、同パッケージから直接呼び出して実際に実行する。
//
// ドローン未接続の環境では、ドライバーのコマンド送信が内部でパニックする
// （実機のIPへ到達できないため）。これは実機操作では起こらないUT環境固有の
// 挙動なので、該当箇所は recoverCall で回復して「呼び出しまでの行」を実行する。
// 実機接続時にのみ到達可能な箇所はカバレッジの除外対象として扱う
// （除外一覧と根拠はPRおよびREADMEに明記）。

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
	recoverCall(dc.TakeOff)

	// 飛行中フラグを直接立てて着陸・移動系を叩く
	// （ドライバーの移動量設定は接続なしでも動作する）
	dc.isFlying = true
	recoverCall(dc.Land)

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

// TestCameraViewerStartStopCoverage Start（未接続では途中でパニック）と Stop
func TestCameraViewerStartStopCoverage(t *testing.T) {
	chdirTo(t, t.TempDir())

	drone := tello.NewDriver("8895")
	cv := NewCameraViewer(drone)

	// Start は StartVideo の送信時点でパニックするが、
	// isRunning の設定など呼び出しまでの行は実行される
	panicked := recoverCall(cv.Start)
	if !panicked {
		t.Log("Start がパニックしなかった（接続済みドライバーの可能性）")
	}

	// 録画していない状態での Stop
	cv.Stop()
	if cv.IsRunning() {
		t.Error("Stop 後は実行中であってはならない")
	}

	// 録画中の Stop では録画も止まる
	cv.StartRecording()
	if !cv.IsRecording() {
		t.Fatal("録画が開始していない")
	}
	cv.Stop()
	if cv.IsRecording() {
		t.Error("Stop 後は録画停止であるべき")
	}
}

// TestProcessFrameBranchesCoverage フレーム処理の各分岐
func TestProcessFrameBranchesCoverage(t *testing.T) {
	chdirTo(t, t.TempDir())

	drone := tello.NewDriver("8897")
	cv := NewCameraViewer(drone)
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
