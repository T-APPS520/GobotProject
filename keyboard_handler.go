package main

import (
	"fmt"
	"os"

	"github.com/nsf/termbox-go"
)

// KeyboardHandler はキーボード入力を処理するクラス
type KeyboardHandler struct {
	droneController *DroneController
	cameraViewer    *CameraViewer
	isRunning       bool
}

// NewKeyboardHandler は新しいキーボードハンドラーを作成
func NewKeyboardHandler(droneController *DroneController, cameraViewer *CameraViewer) *KeyboardHandler {
	return &KeyboardHandler{
		droneController: droneController,
		cameraViewer:    cameraViewer,
		isRunning:       false,
	}
}

// Start はキーボードハンドラーを開始
func (kh *KeyboardHandler) Start() error {
	err := termbox.Init()
	if err != nil {
		return err
	}

	kh.isRunning = true
	fmt.Println("キーボードコントロール開始:")
	fmt.Println("W/A/S/D: 前進/左/後退/右")
	fmt.Println("Space: 上昇")
	fmt.Println("Z: 降下")
	fmt.Println("Escape: 離陸/着陸")
	fmt.Println("L: 録画 開始/停止")
	fmt.Println("Q: 終了")

	go kh.handleKeyboard()
	return nil
}

// Stop はキーボードハンドラーを停止
func (kh *KeyboardHandler) Stop() {
	kh.isRunning = false
	termbox.Close()
}

// handleKeyboard はキーボード入力を処理
func (kh *KeyboardHandler) handleKeyboard() {
	for kh.isRunning {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			kh.processKey(ev)
		case termbox.EventError:
			panic(ev.Err)
		}
	}
}

// processKey はキー入力を処理
func (kh *KeyboardHandler) processKey(ev termbox.Event) {
	switch ev.Key {
	case termbox.KeyEsc:
		// Escapeキー: 離陸/着陸
		kh.droneController.TakeOffOrLand()
		
	case termbox.KeySpace:
		// スペースキー: 上昇
		kh.droneController.MoveUp()
		
	case termbox.KeyCtrlC:
		// Ctrl+C: 終了
		fmt.Println("\nプログラムを終了します...")
		kh.Stop()
		os.Exit(0)
	}

	// 通常のキー入力
	switch ev.Ch {
	case 'w', 'W':
		// W: 前進
		kh.droneController.MoveForward()
		
	case 's', 'S':
		// S: 後退
		kh.droneController.MoveBackward()
		
	case 'a', 'A':
		// A: 左移動
		kh.droneController.MoveLeft()
		
	case 'd', 'D':
		// D: 右移動
		kh.droneController.MoveRight()
		
	case 'l', 'L':
		// L: 録画切り替え
		if kh.cameraViewer != nil {
			kh.cameraViewer.ToggleRecording()
		}
		
	case 'q', 'Q':
		// Q: 終了
		fmt.Println("\nプログラムを終了します...")
		kh.Stop()
		os.Exit(0)
		
	case 'z', 'Z':
		// Z: 降下（Shiftキーの代替）
		kh.droneController.MoveDown()
	}
}

// IsRunning は実行中かどうかを返す
func (kh *KeyboardHandler) IsRunning() bool {
	return kh.isRunning
}
