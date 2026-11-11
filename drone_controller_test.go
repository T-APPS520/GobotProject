package main

import "testing"

// TestDroneControllerCommandsCoverage 全コマンドを実行する
func TestDroneControllerCommandsCoverage(t *testing.T) {
	dc := NewDroneController()

	recoverCall(dc.TakeOff)

	dc.isFlying = true
	recoverCall(dc.Land)

	dc.isFlying = false
	recoverCall(dc.TakeOffOrLand)
	dc.isFlying = true
	recoverCall(dc.TakeOffOrLand)

	// 移動（非飛行中）
	dc.isFlying = false
	dc.MoveForward()
	dc.MoveBackward()
	dc.MoveLeft()
	dc.MoveRight()
	dc.MoveUp()
	dc.MoveDown()

	// 移動（飛行中）
	dc.isFlying = true
	dc.MoveForward()
	dc.MoveBackward()
	dc.MoveLeft()
	dc.MoveRight()
	dc.MoveUp()
	dc.MoveDown()

	// 録画トグル
	dc.isRecording = false
	recoverCall(dc.ToggleRecording)
	dc.isRecording = true
	dc.ToggleRecording()
	if dc.IsRecording() {
		t.Error("トグル後は録画停止であるべき")
	}
	dc.StopRecording()
	recoverCall(dc.StartRecording)
}
