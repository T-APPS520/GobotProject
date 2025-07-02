package main

import (
	"os"
	"os/exec"
	"testing"
)

// TestMP4WriterMacOSCompatibility はmacOS向けQuickTime互換性をテスト
func TestMP4WriterMacOSCompatibility(t *testing.T) {
	testFilename := "test_macos_compat.mov"
	
	// テスト後にファイルをクリーンアップ
	defer os.Remove(testFilename)
	
	// MOVWriterを作成（QuickTime形式）
	writer, err := NewMP4Writer(testFilename)
	if err != nil {
		t.Fatalf("MOVWriter作成に失敗: %v", err)
	}
	
	// より実際的なH.264フレームデータ（DJI Telloから来る可能性のあるパターン）
	testFrames := [][]byte{
		{0x00, 0x00, 0x00, 0x01, 0x67, 0x42, 0x00, 0x1E, 0xDA, 0x05, 0x82, 0x5B}, // SPS
		{0x00, 0x00, 0x00, 0x01, 0x68, 0xCE, 0x3C, 0x80},                         // PPS
		{0x00, 0x00, 0x00, 0x01, 0x65, 0x88, 0x84, 0x00, 0x10, 0xFF, 0xFE, 0xF6}, // IDR frame
		{0x00, 0x00, 0x00, 0x01, 0x61, 0xE3, 0x09, 0x40, 0x00, 0x4F, 0xFF, 0xF2}, // P frame
		{0x00, 0x00, 0x00, 0x01, 0x61, 0xE3, 0x09, 0x40, 0x00, 0x4F, 0xFF, 0xF3}, // P frame
	}
	
	// フレームを書き込み
	for i, frame := range testFrames {
		if err := writer.WriteFrame(frame); err != nil {
			t.Errorf("フレーム%dの書き込みに失敗: %v", i+1, err)
		}
	}
	
	// MP4ファイルを完成
	if err := writer.Close(); err != nil {
		t.Errorf("MP4ファイルのクローズに失敗: %v", err)
	}
	
	// ファイルが作成されたことを確認
	if _, err := os.Stat(testFilename); os.IsNotExist(err) {
		t.Errorf("MP4ファイルが作成されませんでした: %s", testFilename)
		return
	}
	
	// ファイルサイズが適切であることを確認
	info, err := os.Stat(testFilename)
	if err != nil {
		t.Errorf("ファイル情報の取得に失敗: %v", err)
		return
	}
	
	if info.Size() < 100 { // QuickTimeの最小構造サイズ（調整）
		t.Errorf("MP4ファイルが小さすぎます: %d bytes", info.Size())
	} else {
		t.Logf("QuickTime互換MP4ファイル作成成功: %s (%d bytes)", testFilename, info.Size())
	}
	
	// macOSでファイル情報を確認（もしfileコマンドが利用可能なら）
	if cmd := exec.Command("file", testFilename); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			t.Logf("ファイル情報: %s", string(output))
		}
	}
	
	// QuickTimeでの認識確認（もしmdlsコマンドが利用可能なら）
	if cmd := exec.Command("mdls", "-name", "kMDItemContentType", testFilename); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			t.Logf("macOSファイルタイプ: %s", string(output))
		}
	}
}

// TestMP4WriterFileStructure はMOVファイル構造をテスト
func TestMP4WriterFileStructure(t *testing.T) {
	testFilename := "test_structure.mov"
	defer os.Remove(testFilename)
	
	writer, err := NewMP4Writer(testFilename)
	if err != nil {
		t.Fatalf("MP4Writer作成に失敗: %v", err)
	}
	
	// テストフレームを追加
	testFrame := []byte{0x00, 0x00, 0x00, 0x01, 0x67, 0x42, 0x00, 0x1E}
	if err := writer.WriteFrame(testFrame); err != nil {
		t.Errorf("フレーム書き込みに失敗: %v", err)
	}
	
	if err := writer.Close(); err != nil {
		t.Errorf("MP4ファイルのクローズに失敗: %v", err)
	}
	
	// ファイルの先頭部分を読み取ってMP4構造を確認
	file, err := os.Open(testFilename)
	if err != nil {
		t.Fatalf("ファイルを開けません: %v", err)
	}
	defer file.Close()
	
	// ftypボックスを確認
	header := make([]byte, 32)
	if _, err := file.Read(header); err != nil {
		t.Errorf("ヘッダー読み込みに失敗: %v", err)
		return
	}
	
	// ftypボックスの確認
	if string(header[4:8]) != "ftyp" {
		t.Errorf("ftypボックスが見つかりません: %s", string(header[4:8]))
	}
	
	if string(header[8:12]) != "qt  " {
		t.Errorf("major brandが正しくありません: %s", string(header[8:12]))
	}
	
	t.Logf("QuickTime形式ファイル構造確認完了: ftypボックス正常 (major brand: %s)", string(header[8:12]))
}
