package main

import (
	"fmt"
	"log"
	"time"

	"gobot.io/x/gobot"
)

// waitForConnection 接続確認を行い、タイムアウト付きで待機
func waitForConnection(droneController *DroneController, maxWaitTime time.Duration) error {
	log.Println("ドローンに接続中...")
	
	// シンプルな接続確認（実際の実装では、ドローンの状態を確認）
	startTime := time.Now()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	// タイムアウトタイマーをループ外で設定
	timeout := time.After(maxWaitTime)
	
	for {
		select {
		case <-ticker.C:
			// 実際の接続確認ロジック（ここでは簡略化）
			if time.Since(startTime) >= time.Second {
				log.Println("準備完了！")
				return nil
			}
		case <-timeout:
			return fmt.Errorf("接続タイムアウト: %v", maxWaitTime)
		}
	}
}

func main() {
	// ドローンコントローラーを作成
	droneController := NewDroneController()
	
	// カメラビューワーを作成
	cameraViewer := NewCameraViewer(droneController.GetDriver())
	
	// キーボードハンドラーを作成
	keyboardHandler := NewKeyboardHandler(droneController, cameraViewer)

	// ドローンの動作を定義する関数
	work := func() {
		// カメラビューワーを開始
		cameraViewer.Start()
		
		// キーボードハンドラーを開始
		err := keyboardHandler.Start()
		if err != nil {
			log.Printf("キーボードハンドラーの開始に失敗: %v", err)
			return
		}

		// プログラムの説明を表示
		log.Println("=== Tello ドローンコントローラー ===")
		
		// 接続確認のため少し待機
		err = waitForConnection(droneController, 10*time.Second)
		if err != nil {
			log.Printf("接続エラー: %v", err)
			return
		}
	}

	// ロボットを作成し、ドローンデバイスを設定
	robot := gobot.NewRobot(
		[]gobot.Connection{},
		[]gobot.Device{droneController.GetDriver()},
		work,
	)

	// ロボットを開始し、エラーがあれば表示
	err := robot.Start()
	if err != nil {
		log.Printf("ロボット開始エラー: %v", err)
	}
}