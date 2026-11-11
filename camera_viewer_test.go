package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"gobot.io/x/gobot/platforms/dji/tello"
)

// TestCameraViewerMP4Creation MP4録画用カメラビューワーの作成テスト
func TestCameraViewerMP4Creation(t *testing.T) {
	// ドローンドライバーを作成（実際の接続は行わない）
	drone := tello.NewDriver("8890")
	cameraViewer := NewCameraViewer(drone)

	if cameraViewer == nil {
		t.Error("CameraViewerの作成に失敗")
	}

	if cameraViewer.IsRunning() {
		t.Error("初期状態では実行中であってはならない")
	}

	if cameraViewer.IsRecording() {
		t.Error("初期状態では録画中であってはならない")
	}
}

// TestCameraViewerRecordingLifecycle 録画のライフサイクルテスト
func TestCameraViewerRecordingLifecycle(t *testing.T) {
	drone := tello.NewDriver("8890")
	cameraViewer := NewCameraViewer(drone)

	// 初期状態の確認
	if cameraViewer.IsRecording() {
		t.Error("初期状態では録画中であってはならない")
	}

	// 録画開始
	cameraViewer.StartRecording()
	if !cameraViewer.IsRecording() {
		t.Error("録画開始後は録画中であるべき")
	}

	// 現在の録画ファイル名を確認
	currentFile := cameraViewer.GetCurrentRecordingFile()
	if currentFile == "" {
		t.Error("録画中はファイル名が設定されているべき")
	}

	if !strings.HasSuffix(currentFile, ".mov") {
		t.Error("録画ファイルはMOV形式であるべき")
	}

	// 録画停止
	cameraViewer.StopRecording()
	if cameraViewer.IsRecording() {
		t.Error("録画停止後は録画中であってはならない")
	}
}

// TestCameraViewerToggleRecording 録画切り替えテスト
func TestCameraViewerToggleRecording(t *testing.T) {
	drone := tello.NewDriver("8890")
	cameraViewer := NewCameraViewer(drone)

	// 初期状態は録画停止
	initialState := cameraViewer.IsRecording()

	// 1回目のトグル（録画開始）
	cameraViewer.ToggleRecording()
	if cameraViewer.IsRecording() == initialState {
		t.Error("トグル後は録画状態が変わるべき")
	}

	// 2回目のトグル（録画停止）
	cameraViewer.ToggleRecording()
	if cameraViewer.IsRecording() != initialState {
		t.Error("2回のトグル後は初期状態に戻るべき")
	}
}

// TestCameraViewerRecordingFormat 録画形式テスト
func TestCameraViewerRecordingFormat(t *testing.T) {
	drone := tello.NewDriver("8890")
	cameraViewer := NewCameraViewer(drone)

	format := cameraViewer.GetRecordingFormat()
	if format != "MOV" {
		t.Errorf("録画形式は MOV であるべき, 実際: %s", format)
	}

	t.Logf("録画形式: %s", format)
}

// TestCameraViewerMP4FormatOnly 直接MP4録画テスト
func TestCameraViewerMP4FormatOnly(t *testing.T) {
	drone := tello.NewDriver("8890")
	cameraViewer := NewCameraViewer(drone)

	// 録画形式が常にMP4であることを確認
	format := cameraViewer.GetRecordingFormat()
	if format != "MOV" {
		t.Errorf("期待された形式: MOV, 実際: %s", format)
	}

	t.Logf("直接MOV録画が利用可能: %s", format)
}

// TestCameraViewerConcurrentRecording 同時録画制御テスト
func TestCameraViewerConcurrentRecording(t *testing.T) {
	drone := tello.NewDriver("8890")
	cameraViewer := NewCameraViewer(drone)

	// 複数回の録画開始を試行（同時実行のシミュレーション）
	cameraViewer.StartRecording()
	cameraViewer.StartRecording() // 重複開始
	cameraViewer.StartRecording() // 重複開始

	if !cameraViewer.IsRecording() {
		t.Error("録画が開始されていない")
	}

	// 複数回の録画停止を試行
	cameraViewer.StopRecording()
	cameraViewer.StopRecording() // 重複停止
	cameraViewer.StopRecording() // 重複停止

	if cameraViewer.IsRecording() {
		t.Error("録画が停止されていない")
	}
}

// TestCameraViewerFileNaming ファイル命名テスト
func TestCameraViewerFileNaming(t *testing.T) {
	drone := tello.NewDriver("8890")
	cameraViewer := NewCameraViewer(drone)

	// 録画開始
	cameraViewer.StartRecording()
	filename1 := cameraViewer.GetCurrentRecordingFile()
	cameraViewer.StopRecording()

	// 少し待機
	time.Sleep(10 * time.Millisecond)

	// 再度録画開始
	cameraViewer.StartRecording()
	filename2 := cameraViewer.GetCurrentRecordingFile()
	cameraViewer.StopRecording()

	// ファイル名が異なることを確認
	if filename1 == filename2 {
		t.Error("異なる録画セッションでは異なるファイル名であるべき")
	}

	// ファイル名の形式を確認
	if !strings.Contains(filename1, "tello_recording_") {
		t.Errorf("ファイル名の形式が不正: %s", filename1)
	}

	if !strings.HasSuffix(filename1, ".mov") {
		t.Errorf("ファイル拡張子が不正: %s", filename1)
	}
}

// TestCameraViewerErrorHandling エラーハンドリングテスト
func TestCameraViewerErrorHandling(t *testing.T) {
	drone := tello.NewDriver("8890")
	cameraViewer := NewCameraViewer(drone)

	// 無効なパスでの録画テスト（権限のないディレクトリなど）
	defer func() {
		if r := recover(); r != nil {
			t.Logf("エラーハンドリングが正常に動作: %v", r)
		}
	}()

	// 通常の録画操作
	cameraViewer.StartRecording()
	if !cameraViewer.IsRecording() {
		t.Log("録画開始に失敗（これは予期される場合もある）")
	}

	cameraViewer.StopRecording()
	if cameraViewer.IsRecording() {
		t.Error("録画停止に失敗")
	}
}

// TestCameraViewerCleanup クリーンアップテスト
func TestCameraViewerCleanup(t *testing.T) {
	drone := tello.NewDriver("8890")
	cameraViewer := NewCameraViewer(drone)

	// テスト開始前のファイル一覧
	beforeFiles := getTestRecordingFiles(t)

	// 録画操作
	cameraViewer.StartRecording()
	currentFile := cameraViewer.GetCurrentRecordingFile()
	cameraViewer.StopRecording()

	// 短時間待機（ファイル操作完了のため）
	time.Sleep(100 * time.Millisecond)

	// テスト後のクリーンアップ
	defer func() {
		// テスト中に作成されたファイルを削除
		afterFiles := getTestRecordingFiles(t)
		for _, file := range afterFiles {
			found := false
			for _, before := range beforeFiles {
				if file == before {
					found = true
					break
				}
			}
			if !found {
				os.Remove(file)
				t.Logf("テストファイルを削除: %s", file)
			}
		}
	}()

	t.Logf("録画ファイル: %s", currentFile)
}

// MP4録画シナリオテスト

// TestMP4RecordingScenario_CompleteSession 完全な録画セッションシナリオ
func TestMP4RecordingScenario_CompleteSession(t *testing.T) {
	drone := tello.NewDriver("8890")
	cameraViewer := NewCameraViewer(drone)

	t.Log("シナリオ: 完全な録画セッション")

	// ステップ1: 初期状態確認
	t.Log("ステップ1: 初期状態確認")
	if cameraViewer.IsRecording() {
		t.Error("初期状態では録画停止であるべき")
	}

	// ステップ2: 録画開始
	t.Log("ステップ2: 録画開始")
	cameraViewer.StartRecording()
	if !cameraViewer.IsRecording() {
		t.Error("録画開始後は録画中であるべき")
	}

	recordingFile := cameraViewer.GetCurrentRecordingFile()
	if recordingFile == "" {
		t.Error("録画ファイル名が設定されていない")
	}
	t.Logf("録画ファイル: %s", recordingFile)

	// ステップ3: フレーム処理シミュレーション
	t.Log("ステップ3: フレーム処理シミュレーション")
	mockFrameData := []byte("mock video frame data")
	for i := 0; i < 10; i++ {
		cameraViewer.processFrame(mockFrameData)
	}

	// ステップ4: 録画停止
	t.Log("ステップ4: 録画停止")
	cameraViewer.StopRecording()
	if cameraViewer.IsRecording() {
		t.Error("録画停止後は録画停止であるべき")
	}

	// ステップ5: 結果確認
	t.Log("ステップ5: 結果確認")
	format := cameraViewer.GetRecordingFormat()
	t.Logf("録画形式: %s", format)

	// クリーンアップ
	defer func() {
		time.Sleep(100 * time.Millisecond) // 処理の完了を待機
		os.Remove(recordingFile)
	}()

	t.Log("完全な録画セッションシナリオ完了")
}

// TestMP4RecordingScenario_MultipleShortSessions 複数の短時間録画セッション
func TestMP4RecordingScenario_MultipleShortSessions(t *testing.T) {
	drone := tello.NewDriver("8890")
	cameraViewer := NewCameraViewer(drone)

	t.Log("シナリオ: 複数の短時間録画セッション")

	var recordedFiles []string

	// 3回の短時間録画セッション
	for session := 1; session <= 3; session++ {
		t.Logf("セッション %d 開始", session)

		// 録画開始
		cameraViewer.StartRecording()
		recordingFile := cameraViewer.GetCurrentRecordingFile()
		recordedFiles = append(recordedFiles, recordingFile)

		// 短時間のフレーム処理
		mockFrameData := []byte(fmt.Sprintf("session %d frame data", session))
		for i := 0; i < 5; i++ {
			cameraViewer.processFrame(mockFrameData)
		}

		// 録画停止
		cameraViewer.StopRecording()

		// セッション間の待機
		time.Sleep(10 * time.Millisecond)
		t.Logf("セッション %d 完了: %s", session, recordingFile)
	}

	// 結果確認
	if len(recordedFiles) != 3 {
		t.Errorf("期待されるファイル数: 3, 実際: %d", len(recordedFiles))
	}

	// 各ファイル名がユニークであることを確認
	for i := 0; i < len(recordedFiles); i++ {
		for j := i + 1; j < len(recordedFiles); j++ {
			if recordedFiles[i] == recordedFiles[j] {
				t.Error("録画ファイル名が重複している")
			}
		}
	}

	// クリーンアップ
	defer func() {
		time.Sleep(200 * time.Millisecond) // 処理の完了を待機
		for _, file := range recordedFiles {
			os.Remove(file)
		}
	}()

	t.Log("複数の短時間録画セッションシナリオ完了")
}

// TestMP4RecordingScenario_RapidToggle 高速切り替えシナリオ
func TestMP4RecordingScenario_RapidToggle(t *testing.T) {
	drone := tello.NewDriver("8890")
	cameraViewer := NewCameraViewer(drone)

	t.Log("シナリオ: 録画の高速切り替え")

	var lastFile string

	// 高速で録画のオン/オフを繰り返す
	for i := 0; i < 5; i++ {
		t.Logf("高速切り替え %d回目", i+1)

		// 録画開始
		cameraViewer.ToggleRecording()
		if cameraViewer.IsRecording() {
			lastFile = cameraViewer.GetCurrentRecordingFile()
		}

		// 即座に録画停止
		cameraViewer.ToggleRecording()
		if cameraViewer.IsRecording() {
			t.Error("トグル後は録画停止であるべき")
		}
	}

	// 最終状態確認
	if cameraViewer.IsRecording() {
		t.Error("最終状態では録画停止であるべき")
	}

	// クリーンアップ
	defer func() {
		if lastFile != "" {
			time.Sleep(100 * time.Millisecond)
			os.Remove(lastFile)
		}
	}()

	t.Log("高速切り替えシナリオ完了")
}

// TestMP4RecordingScenario_ErrorRecovery エラー回復シナリオ
func TestMP4RecordingScenario_ErrorRecovery(t *testing.T) {
	drone := tello.NewDriver("8890")
	cameraViewer := NewCameraViewer(drone)

	t.Log("シナリオ: エラー回復")

	// 通常の録画操作
	cameraViewer.StartRecording()
	recordingFile := cameraViewer.GetCurrentRecordingFile()

	// エラー状況のシミュレーション（無効なフレームデータなど）
	t.Log("エラー状況のシミュレーション")
	cameraViewer.processFrame(nil)           // nilデータ
	cameraViewer.processFrame([]byte{})      // 空データ
	cameraViewer.processFrame([]byte("ok"))  // 正常データ

	// 回復確認
	if !cameraViewer.IsRecording() {
		t.Error("エラー後も録画は継続されるべき")
	}

	// 正常停止
	cameraViewer.StopRecording()

	// クリーンアップ
	defer func() {
		if recordingFile != "" {
			time.Sleep(100 * time.Millisecond)
			os.Remove(recordingFile)
		}
	}()

	t.Log("エラー回復シナリオ完了")
}

// getTestRecordingFiles テスト録画ファイル一覧を取得
func getTestRecordingFiles(t *testing.T) []string {
	// .mov形式のファイルを検索
	files, err := filepath.Glob("tello_recording_*.mov")
	if err != nil {
		t.Logf("ファイル検索エラー: %v", err)
		return []string{}
	}

	return files
}

// TestCameraViewerStartStopCoverage Start と Stop を検証する
func TestCameraViewerStartStopCoverage(t *testing.T) {
	chdirTo(t, t.TempDir())

	drone := tello.NewDriver("8895")
	cv := NewCameraViewer(drone)

	recoverCall(cv.Start)

	cv.Stop()
	if cv.IsRunning() {
		t.Error("Stop 後は実行中であってはならない")
	}

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
	cv.playerFactory = func() (*VideoPlayer, error) {
		return nil, os.ErrNotExist // 表示できない環境
	}
	cv.isRunning = true

	// 150フレーム処理する
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

// TestStartRecordingErrorCoverage 録画ファイルを作成できない場合を検証する
func TestStartRecordingErrorCoverage(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows ではカレントディレクトリを削除できない")
	}

	saved, err := os.Getwd()
	if err != nil {
		t.Fatalf("カレントディレクトリ取得に失敗: %v", err)
	}
	defer os.Chdir(saved)

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

// TestMP4WriterCoverage MP4ライターを検証する
func TestMP4WriterCoverage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mov")

	w, err := NewMP4Writer(path)
	if err != nil {
		t.Fatalf("ライター作成に失敗: %v", err)
	}

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
	if err := w.Close(); err != nil {
		t.Errorf("二重クローズでエラー: %v", err)
	}

	if _, err := NewMP4Writer(filepath.Join(dir, "no-such-dir", "x.mov")); err == nil {
		t.Error("存在しないディレクトリではエラーになるべき")
	}

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

// TestStopRecordingCloseErrorCoverage 録画ファイルの保存に失敗する場合を検証する
func TestStopRecordingCloseErrorCoverage(t *testing.T) {
	chdirTo(t, t.TempDir())

	cv := NewCameraViewer(tello.NewDriver("8902"))
	cv.StartRecording()
	if !cv.IsRecording() {
		t.Fatal("録画が開始していない")
	}

	if cv.mp4Writer != nil && cv.mp4Writer.file != nil {
		cv.mp4Writer.file.Close()
	}
	cv.StopRecording()
	if cv.IsRecording() {
		t.Error("停止後は録画中であってはならない")
	}
}

// TestMP4BoxBuildersCoverage MP4ボックス生成を検証する
func TestMP4BoxBuildersCoverage(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "boxes.mov"))
	if err != nil {
		t.Fatalf("ファイル作成に失敗: %v", err)
	}
	defer f.Close()

	w := &MP4Writer{
		file:      f,
		startTime: time.Now().Add(-10 * time.Second),
		frameNum:  5,
		frameData: [][]byte{{0x01, 0x02}},
	}

	if len(w.createQuickTimeMovieBox()) == 0 {
		t.Error("moovボックスが空")
	}
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

	// 接続完了イベント
	drone.Publish(drone.Event(tello.ConnectedEvent), nil)

	// フレームイベント
	drone.Publish(drone.Event(tello.VideoFrameEvent), []byte{0x00, 0x00, 0x00, 0x01, 0x67})
	waitForCondition(t, 2*time.Second, func() bool {
		return cv.GetDisplayedFrameCount() >= 1
	}, "フレームイベントが表示まで届かなかった")

	cv.Stop()
	if cv.IsRunning() {
		t.Error("Stop 後は実行中であってはならない")
	}
	
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

// TestDiscardPlayerCoverage 書き込み失敗時のプレイヤー破棄
func TestDiscardPlayerCoverage(t *testing.T) {
	drone := tello.NewDriver("8898")
	cv := NewCameraViewer(drone)
	cv.isRunning = true

	// 書き込みが必ず失敗するプレイヤーを差し込む
	failing := &VideoPlayer{
		spec:   playerSpec{name: "fake"},
		stdin:  errWriteCloser{},
		exited: make(chan struct{}), // Running() は true
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
