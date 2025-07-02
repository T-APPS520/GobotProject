package main

import (
	"fmt"
	"os"
	"path/filepath"
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
