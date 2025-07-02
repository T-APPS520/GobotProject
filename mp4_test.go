package main

import (
	"os"
	"testing"
	"time"
)

// TestMP4WriterDirectRecording は直接MOV録画機能をテスト
func TestMP4WriterDirectRecording(t *testing.T) {
	testFilename := "test_direct_mov.mov"
	
	// テスト後にファイルをクリーンアップ
	defer os.Remove(testFilename)
	
	// MOVWriterを作成（QuickTime形式）
	writer, err := NewMP4Writer(testFilename)
	if err != nil {
		t.Fatalf("MOVWriter作成に失敗: %v", err)
	}
	
	// テスト用のH.264フレームデータ（ダミー）
	testFrame1 := []byte{0x00, 0x00, 0x00, 0x01, 0x67, 0x42, 0x00, 0x1E} // SPS
	testFrame2 := []byte{0x00, 0x00, 0x00, 0x01, 0x68, 0xCE, 0x3C, 0x80} // PPS
	testFrame3 := []byte{0x00, 0x00, 0x00, 0x01, 0x65, 0x88, 0x84, 0x00} // IDR frame
	
	// フレームを書き込み
	if err := writer.WriteFrame(testFrame1); err != nil {
		t.Errorf("フレーム1の書き込みに失敗: %v", err)
	}
	
	if err := writer.WriteFrame(testFrame2); err != nil {
		t.Errorf("フレーム2の書き込みに失敗: %v", err)
	}
	
	if err := writer.WriteFrame(testFrame3); err != nil {
		t.Errorf("フレーム3の書き込みに失敗: %v", err)
	}
	
	// MP4ファイルを完成
	if err := writer.Close(); err != nil {
		t.Errorf("MP4ファイルのクローズに失敗: %v", err)
	}
	
	// ファイルが作成されたことを確認
	if _, err := os.Stat(testFilename); os.IsNotExist(err) {
		t.Errorf("MP4ファイルが作成されませんでした: %s", testFilename)
	}
	
	// ファイルサイズが0より大きいことを確認
	info, err := os.Stat(testFilename)
	if err != nil {
		t.Errorf("ファイル情報の取得に失敗: %v", err)
	} else if info.Size() == 0 {
		t.Errorf("MP4ファイルのサイズが0です")
	} else {
		t.Logf("MOVファイル作成成功: %s (%d bytes)", testFilename, info.Size())
	}
}

// TestCameraViewerDirectMP4Recording はCameraViewerの直接MP4録画をテスト
func TestCameraViewerDirectMP4Recording(t *testing.T) {
	// ドローンコントローラーを作成（モック）
	droneController := NewDroneController()
	cameraViewer := NewCameraViewer(droneController.GetDriver())
	
	// 録画開始
	cameraViewer.StartRecording()
	
	if !cameraViewer.IsRecording() {
		t.Error("録画が開始されていません")
	}
	
	// 録画形式がMP4であることを確認
	format := cameraViewer.GetRecordingFormat()
	if format != "MOV" {
		t.Errorf("期待される録画形式: MOV, 実際: %s", format)
	}
	
	// 録画ファイル名が設定されていることを確認
	filename := cameraViewer.GetCurrentRecordingFile()
	if filename == "" {
		t.Error("録画ファイル名が設定されていません")
	}
	
	// テスト用フレームデータを送信
	testFrame := []byte{0x00, 0x00, 0x00, 0x01, 0x67, 0x42, 0x00, 0x1E}
	cameraViewer.processFrame(testFrame)
	
	// 少し待ってから録画停止
	time.Sleep(100 * time.Millisecond)
	cameraViewer.StopRecording()
	
	if cameraViewer.IsRecording() {
		t.Error("録画が停止されていません")
	}
	
	// 録画ファイルが作成されたことを確認
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Errorf("録画ファイルが作成されませんでした: %s", filename)
	} else {
		// テスト後にファイルをクリーンアップ
		defer os.Remove(filename)
	}
}

// TestMP4WriterEmptyFrame は空フレームの処理をテスト
func TestMP4WriterEmptyFrame(t *testing.T) {
	testFilename := "test_empty_frame.mp4"
	defer os.Remove(testFilename)
	
	writer, err := NewMP4Writer(testFilename)
	if err != nil {
		t.Fatalf("MP4Writer作成に失敗: %v", err)
	}
	
	// 空フレームを書き込み（エラーにならないことを確認）
	if err := writer.WriteFrame([]byte{}); err != nil {
		t.Errorf("空フレームの書き込みでエラー: %v", err)
	}
	
	if err := writer.WriteFrame(nil); err != nil {
		t.Errorf("nilフレームの書き込みでエラー: %v", err)
	}
	
	if err := writer.Close(); err != nil {
		t.Errorf("MP4ファイルのクローズに失敗: %v", err)
	}
}
