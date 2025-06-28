package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"gobot.io/x/gobot/platforms/dji/tello"
)

// CameraViewer はドローンのカメラ画像を表示するクラス
type CameraViewer struct {
	drone       *tello.Driver
	isRunning   bool
	isRecording bool
	frameCount  int
	recordFile  *os.File
}

// NewCameraViewer は新しいカメラビューワーを作成
func NewCameraViewer(drone *tello.Driver) *CameraViewer {
	return &CameraViewer{
		drone:       drone,
		isRunning:   false,
		isRecording: false,
		frameCount:  0,
	}
}

// Start はカメラビューワーを開始
func (cv *CameraViewer) Start() {
	cv.isRunning = true
	
	// ビデオストリームを開始
	cv.drone.StartVideo()
	cv.drone.SetVideoEncoderRate(tello.VideoBitRateAuto)
	cv.drone.SetExposure(0)

	// ビデオフレームイベントを登録
	cv.drone.On(tello.VideoFrameEvent, func(data interface{}) {
		if frameData, ok := data.([]byte); ok {
			cv.processFrame(frameData)
		}
	})

	fmt.Println("カメラビューワー開始 - ビデオストリーム受信中...")
}

// Stop はカメラビューワーを停止
func (cv *CameraViewer) Stop() {
	cv.isRunning = false
	
	if cv.isRecording {
		cv.StopRecording()
	}
	
	fmt.Println("カメラビューワー停止")
}

// processFrame はフレームを処理
func (cv *CameraViewer) processFrame(frameData []byte) {
	if !cv.isRunning {
		return
	}

	cv.frameCount++
	
	// フレーム受信の確認（5秒ごと）
	if cv.frameCount%150 == 0 { // 約30FPS * 5秒
		fmt.Printf("フレーム受信中... (フレーム数: %d)\n", cv.frameCount)
	}

	// 録画中の場合、フレームデータを保存
	if cv.isRecording && cv.recordFile != nil {
		cv.recordFile.Write(frameData)
	}
}

// StartRecording は録画を開始
func (cv *CameraViewer) StartRecording() {
	if cv.isRecording {
		return
	}

	// 現在の時刻でファイル名を生成
	filename := fmt.Sprintf("tello_recording_%s.h264", time.Now().Format("20060102_150405"))
	
	// ファイルを作成
	file, err := os.Create(filename)
	if err != nil {
		log.Printf("録画ファイルの作成に失敗: %v", err)
		return
	}

	cv.recordFile = file
	cv.isRecording = true
	fmt.Printf("録画開始: %s\n", filename)
}

// StopRecording は録画を停止
func (cv *CameraViewer) StopRecording() {
	if !cv.isRecording {
		return
	}

	if cv.recordFile != nil {
		cv.recordFile.Close()
		cv.recordFile = nil
	}

	cv.isRecording = false
	fmt.Println("録画停止")
}

// ToggleRecording は録画のオン/オフを切り替える
func (cv *CameraViewer) ToggleRecording() {
	if cv.isRecording {
		cv.StopRecording()
	} else {
		cv.StartRecording()
	}
}

// IsRecording は録画中かどうかを返す
func (cv *CameraViewer) IsRecording() bool {
	return cv.isRecording
}

// IsRunning は実行中かどうかを返す
func (cv *CameraViewer) IsRunning() bool {
	return cv.isRunning
}
