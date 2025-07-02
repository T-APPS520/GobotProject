package main

import (
	"os"
	"testing"
)

// TestMP4WriterOnly は新しいMP4Writer機能のみをテスト
func TestMP4WriterOnly(t *testing.T) {
	testFilename := "test_new_mov_writer.mov"
	
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
		t.Errorf("MOVファイルのサイズが0です")
	} else {
		t.Logf("新しいMOVWriter機能: ファイル作成成功 %s (%d bytes)", testFilename, info.Size())
	}
}
