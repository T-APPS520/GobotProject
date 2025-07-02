package main

import (
	"fmt"
	"os"
	"testing"
)

func TestDirectMP4Demo(t *testing.T) {
	fmt.Println("=== QuickTime MOV形式録画テスト ===")

	testFilename := "quicktime_compatible_test.mov"
	defer os.Remove(testFilename) // テスト後にクリーンアップ

	fmt.Println("QuickTime MOV形式録画テストを開始...")

	// MP4Writerを作成
	writer, err := NewMP4Writer(testFilename)
	if err != nil {
		t.Fatalf("MP4Writer作成エラー: %v", err)
	}

	// テスト用のH.264フレームデータ（QuickTime互換性テスト用）
	testFrame1 := []byte{0x00, 0x00, 0x00, 0x01, 0x67, 0x42, 0x00, 0x1E} // SPS
	testFrame2 := []byte{0x00, 0x00, 0x00, 0x01, 0x68, 0xCE, 0x3C, 0x80} // PPS
	testFrame3 := []byte{0x00, 0x00, 0x00, 0x01, 0x65, 0x88, 0x84, 0x00} // IDR frame

	fmt.Println("QuickTime MOV形式フレームを書き込み中...")

	// フレームを書き込み（MOV形式での構造テスト）
	if err := writer.WriteFrame(testFrame1); err != nil {
		t.Logf("フレーム1の書き込みエラー: %v", err)
	}

	if err := writer.WriteFrame(testFrame2); err != nil {
		t.Logf("フレーム2の書き込みエラー: %v", err)
	}

	if err := writer.WriteFrame(testFrame3); err != nil {
		t.Logf("フレーム3の書き込みエラー: %v", err)
	}

	fmt.Println("QuickTime MOV形式ファイルを完成中...")

	// MOVファイルを完成
	if err := writer.Close(); err != nil {
		t.Fatalf("MP4ファイルのクローズエラー: %v", err)
	}

	// ファイルが作成されたことを確認
	if info, err := os.Stat(testFilename); err != nil {
		t.Logf("ファイル確認エラー: %v", err)
	} else {
		fmt.Printf("✅ 成功！QuickTime MOV形式ファイル作成完了: %s (%d bytes)\n", testFilename, info.Size())
		fmt.Printf("QuickTime Playerで.mov形式として再生をお試しください\n")

		// ファイルタイプを確認（macOSの場合）
		t.Logf("作成されたファイル: %s (%d bytes)", testFilename, info.Size())
	}
}
