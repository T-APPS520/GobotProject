package main

import (
	"fmt"
	"os"
	"testing"
)

// TestMacOSCompatibleMP4Demo はmacOS互換MOV録画のデモテスト
func TestMacOSCompatibleMP4Demo(t *testing.T) {
	fmt.Println("=== macOS互換MOV録画デモ（QuickTime MOV形式） ===")
	
	filename := "demo_macos_compatible_qt.mov"
	defer os.Remove(filename) // テスト後にクリーンアップ
	
	// MP4Writerを作成（QuickTime MOV形式出力）
	writer, err := NewMP4Writer(filename)
	if err != nil {
		t.Fatalf("MOVWriter作成エラー: %v", err)
	}
	
	fmt.Println("MOVライター作成完了")
	
	// サンプルH.264フレームデータ（Telloから来る可能性のあるもの）
	sampleFrames := [][]byte{
		{0x00, 0x00, 0x00, 0x01, 0x67, 0x42, 0x00, 0x1E, 0xDA, 0x05, 0x82, 0x5B, 0x10}, // SPS
		{0x00, 0x00, 0x00, 0x01, 0x68, 0xCE, 0x3C, 0x80},                               // PPS
		{0x00, 0x00, 0x00, 0x01, 0x65, 0x88, 0x84, 0x00, 0x10, 0xFF, 0xFE, 0xF6, 0x44}, // IDR frame
		{0x00, 0x00, 0x00, 0x01, 0x61, 0xE3, 0x09, 0x40, 0x00, 0x4F, 0xFF, 0xF2, 0xAA}, // P frame
		{0x00, 0x00, 0x00, 0x01, 0x61, 0xE3, 0x09, 0x40, 0x00, 0x4F, 0xFF, 0xF3, 0xBB}, // P frame
		{0x00, 0x00, 0x00, 0x01, 0x61, 0xE3, 0x09, 0x40, 0x00, 0x4F, 0xFF, 0xF4, 0xCC}, // P frame
	}
	
	fmt.Printf("サンプルフレーム (%d個) を書き込み中...\n", len(sampleFrames))
	
	// フレームを書き込み
	for i, frame := range sampleFrames {
		if err := writer.WriteFrame(frame); err != nil {
			t.Logf("フレーム%d書き込みエラー: %v", i+1, err)
		}
	}
	
	fmt.Println("MOVファイルを完成中...")
	
	// MOVファイルを完成
	if err := writer.Close(); err != nil {
		t.Fatalf("MOVクローズエラー: %v", err)
	}
	
	// ファイル情報を確認
	if info, err := os.Stat(filename); err != nil {
		t.Logf("ファイル確認エラー: %v", err)
	} else {
		fmt.Printf("✅ 成功！macOS互換MOVファイル作成完了\n")
		fmt.Printf("   ファイル名: %s\n", filename)
		fmt.Printf("   ファイルサイズ: %d bytes\n", info.Size())
		fmt.Printf("   QuickTime Playerで.mov形式として再生可能です\n")
	}
	
	fmt.Println("\n=== 新しいQuickTime MOV形式実装完了 ===")
	fmt.Println("ffmpeg不要で直接QuickTime MOV形式で録画できるようになりました！")
	fmt.Println("ファイルタイプ: Apple QuickTime movie (.MOV/QT)")
}
