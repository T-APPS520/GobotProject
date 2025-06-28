package main

import (
	"fmt"
	"time"

	"gobot.io/x/gobot"
)

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
			fmt.Printf("キーボードハンドラーの開始に失敗: %v\n", err)
			return
		}

		// プログラムの説明を表示
		fmt.Println("=== Tello ドローンコントローラー ===")
		fmt.Println("ドローンに接続中...")
		
		// 接続確認のため少し待機
		time.Sleep(2 * time.Second)
		fmt.Println("準備完了！")
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
		fmt.Println("Error starting robot:", err)
	}
}