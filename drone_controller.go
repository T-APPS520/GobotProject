package main

import (
	"fmt"
	"gobot.io/x/gobot/platforms/dji/tello"
)

// DroneController はTelloドローンを制御するクラス
type DroneController struct {
	drone      *tello.Driver
	isFlying   bool
	isRecording bool
}

// NewDroneController は新しいドローンコントローラーを作成
func NewDroneController() *DroneController {
	drone := tello.NewDriver("8888")
	return &DroneController{
		drone:      drone,
		isFlying:   false,
		isRecording: false,
	}
}

// GetDriver はドローンドライバーを返す
func (dc *DroneController) GetDriver() *tello.Driver {
	return dc.drone
}

// TakeOffOrLand は離陸または着陸を制御
func (dc *DroneController) TakeOffOrLand() {
	if dc.isFlying {
		dc.Land()
	} else {
		dc.TakeOff()
	}
}

// TakeOff はドローンを離陸させる
func (dc *DroneController) TakeOff() {
	fmt.Println("ドローンが離陸します...")
	dc.drone.TakeOff()
	dc.isFlying = true
}

// Land はドローンを着陸させる
func (dc *DroneController) Land() {
	fmt.Println("ドローンが着陸します...")
	dc.drone.Land()
	dc.isFlying = false
}

// MoveForward はドローンを前進させる
func (dc *DroneController) MoveForward() {
	if dc.isFlying {
		fmt.Println("前進")
		dc.drone.Forward(20)
	}
}

// MoveBackward はドローンを後退させる
func (dc *DroneController) MoveBackward() {
	if dc.isFlying {
		fmt.Println("後退")
		dc.drone.Backward(20)
	}
}

// MoveLeft はドローンを左に移動させる
func (dc *DroneController) MoveLeft() {
	if dc.isFlying {
		fmt.Println("左移動")
		dc.drone.Left(20)
	}
}

// MoveRight はドローンを右に移動させる
func (dc *DroneController) MoveRight() {
	if dc.isFlying {
		fmt.Println("右移動")
		dc.drone.Right(20)
	}
}

// MoveUp はドローンを上昇させる
func (dc *DroneController) MoveUp() {
	if dc.isFlying {
		fmt.Println("上昇")
		dc.drone.Up(20)
	}
}

// MoveDown はドローンを降下させる
func (dc *DroneController) MoveDown() {
	if dc.isFlying {
		fmt.Println("降下")
		dc.drone.Down(20)
	}
}

// ToggleRecording は録画のオン/オフを切り替える
func (dc *DroneController) ToggleRecording() {
	if dc.isRecording {
		dc.StopRecording()
	} else {
		dc.StartRecording()
	}
}

// StartRecording は録画を開始（ビデオストリーム）
func (dc *DroneController) StartRecording() {
	fmt.Println("録画開始")
	dc.drone.StartVideo()
	dc.isRecording = true
}

// StopRecording は録画を停止
func (dc *DroneController) StopRecording() {
	fmt.Println("録画停止")
	// ビデオストリームは手動で停止しない（カメラビューワー側で制御）
	dc.isRecording = false
}

// IsFlying はドローンが飛行中かどうかを返す
func (dc *DroneController) IsFlying() bool {
	return dc.isFlying
}

// IsRecording はドローンが録画中かどうかを返す
func (dc *DroneController) IsRecording() bool {
	return dc.isRecording
}
