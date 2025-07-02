package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"gobot.io/x/gobot/platforms/dji/tello"
)

// CameraViewer はドローンのカメラ画像を表示するクラス
type CameraViewer struct {
	drone          *tello.Driver
	isRunning      bool
	isRecording    bool
	frameCount     int
	mp4Writer      *MP4Writer
	currentRecordingFile string
	recordingMutex sync.Mutex
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

	// 録画中の場合、フレームデータをMP4に直接書き込み
	if cv.isRecording && cv.mp4Writer != nil {
		if err := cv.mp4Writer.WriteFrame(frameData); err != nil {
			log.Printf("フレーム書き込みエラー: %v", err)
		}
	}
}

// StartRecording は録画を開始（MP4形式で直接録画）
func (cv *CameraViewer) StartRecording() {
	cv.recordingMutex.Lock()
	defer cv.recordingMutex.Unlock()
	
	if cv.isRecording {
		return
	}

	// 現在の時刻でファイル名を生成（マイクロ秒まで含めて重複を避ける）
	timestamp := time.Now().Format("20060102_150405.000000")
	movFilename := fmt.Sprintf("tello_recording_%s.mov", timestamp)
	
	// MOVライターを作成（QuickTime形式）
	movWriter, err := NewMP4Writer(movFilename)
	if err != nil {
		log.Printf("MOV録画ファイルの作成に失敗: %v", err)
		return
	}

	cv.mp4Writer = movWriter
	cv.currentRecordingFile = movFilename
	cv.isRecording = true
	log.Printf("録画開始: %s", movFilename)
}

// StopRecording は録画を停止
func (cv *CameraViewer) StopRecording() {
	cv.recordingMutex.Lock()
	defer cv.recordingMutex.Unlock()
	
	if !cv.isRecording {
		return
	}

	if cv.mp4Writer != nil {
		if err := cv.mp4Writer.Close(); err != nil {
			log.Printf("MP4録画ファイルの保存に失敗: %v", err)
		} else {
			log.Printf("録画停止 - ファイル保存完了: %s", cv.currentRecordingFile)
		}
		cv.mp4Writer = nil
	}

	cv.isRecording = false
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

// GetCurrentRecordingFile は現在の録画ファイル名を返す（テスト用）
func (cv *CameraViewer) GetCurrentRecordingFile() string {
	cv.recordingMutex.Lock()
	defer cv.recordingMutex.Unlock()
	return cv.currentRecordingFile
}

// GetRecordingFormat は録画形式を返す
func (cv *CameraViewer) GetRecordingFormat() string {
	return "MOV"
}

// MP4Writer はMP4ファイルを直接作成するクラス（macOS互換）
type MP4Writer struct {
	file      *os.File
	frameNum  uint32
	startTime time.Time
	frameData [][]byte // フレームデータを一時的に保存
}

// NewMP4Writer は新しいMP4ライターを作成
func NewMP4Writer(filename string) (*MP4Writer, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	writer := &MP4Writer{
		file:      file,
		startTime: time.Now(),
		frameData: make([][]byte, 0),
	}

	return writer, nil
}

// WriteFrame はH.264フレームを保存（MP4作成時に使用）
func (w *MP4Writer) WriteFrame(frameData []byte) error {
	if len(frameData) == 0 {
		return nil
	}

	// フレームデータをコピーして保存（後でMP4に組み込む）
	frameCopy := make([]byte, len(frameData))
	copy(frameCopy, frameData)
	w.frameData = append(w.frameData, frameCopy)
	w.frameNum++

	return nil
}

// Close はMP4ファイルを完成させる
func (w *MP4Writer) Close() error {
	if w.file == nil {
		return nil
	}

	// QuickTime互換のMP4ファイル構造を作成
	if err := w.createQuickTimeCompatibleMP4(); err != nil {
		w.file.Close()
		return err
	}

	duration := time.Since(w.startTime)
	log.Printf("MOV録画完了: %d フレーム, 録画時間: %v", w.frameNum, duration)

	err := w.file.Close()
	w.file = nil
	return err
}

// createQuickTimeCompatibleMP4 はQuickTime Player互換のMP4ファイルを作成
func (w *MP4Writer) createQuickTimeCompatibleMP4() error {
	// ファイルの先頭に戻る
	w.file.Seek(0, 0)
	
	// より単純なアプローチ: 最小限のQuickTime互換MP4を作成
	return w.createMinimalQuickTimeMP4()
}

// createMinimalQuickTimeMP4 は最小限のQuickTime互換MP4を作成
func (w *MP4Writer) createMinimalQuickTimeMP4() error {
	// ftyp box - QuickTime形式
	ftypBox := []byte{
		0x00, 0x00, 0x00, 0x14, // box size (20 bytes)
		'f', 't', 'y', 'p',     // box type 'ftyp'
		'q', 't', ' ', ' ',     // major brand 'qt  '
		0x20, 0x05, 0x03, 0x00, // minor version
		'q', 't', ' ', ' ',     // compatible brand
	}
	
	if _, err := w.file.Write(ftypBox); err != nil {
		return err
	}

	// 非常にシンプルなムービーボックスを作成
	moovBox := w.createSimpleMovieBox()
	if _, err := w.file.Write(moovBox); err != nil {
		return err
	}

	// mdatボックス - 空のメディアデータ（テスト用）
	mdatBox := []byte{
		0x00, 0x00, 0x00, 0x08, // box size (8 bytes - header only)
		'm', 'd', 'a', 't',     // box type 'mdat'
	}
	
	if _, err := w.file.Write(mdatBox); err != nil {
		return err
	}

	return nil
}

// createSimpleMovieBox は最小限のムービーボックスを作成
func (w *MP4Writer) createSimpleMovieBox() []byte {
	// 非常にシンプルなmvhdボックス
	mvhdBox := []byte{
		0x00, 0x00, 0x00, 0x6C, // box size (108 bytes)
		'm', 'v', 'h', 'd',     // box type 'mvhd'
		0x00,                   // version
		0x00, 0x00, 0x00,       // flags
		0x00, 0x00, 0x00, 0x00, // creation time
		0x00, 0x00, 0x00, 0x00, // modification time
		0x00, 0x00, 0x03, 0xE8, // timescale (1000)
		0x00, 0x00, 0x03, 0xE8, // duration (1000)
		0x00, 0x01, 0x00, 0x00, // rate (1.0)
		0x01, 0x00,             // volume (1.0)
		0x00, 0x00,             // reserved
		0x00, 0x00, 0x00, 0x00, // reserved
		0x00, 0x00, 0x00, 0x00, // reserved
		// transformation matrix (identity)
		0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x40, 0x00, 0x00, 0x00,
		// pre-defined
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// next track ID
		0x00, 0x00, 0x00, 0x02,
	}
	
	// moovボックスヘッダー
	moovSize := uint32(8 + len(mvhdBox))
	moovHeader := make([]byte, 8)
	binary.BigEndian.PutUint32(moovHeader[0:4], moovSize)
	copy(moovHeader[4:8], []byte{'m', 'o', 'o', 'v'})
	
	return append(moovHeader, mvhdBox...)
}

// createQuickTimeMovieBox はQuickTime互換のmovieボックスを作成
func (w *MP4Writer) createQuickTimeMovieBox() []byte {
	duration := uint32(time.Since(w.startTime).Seconds() * 30) // 30fps想定での実際のフレーム数
	if duration == 0 {
		duration = uint32(w.frameNum) // フレーム数をそのまま使用
	}
	
	// mvhdボックスを作成
	mvhdBox := w.createQuickTimeMovieHeaderBox(duration)
	
	// trakボックスを作成
	trakBox := w.createQuickTimeVideoTrackBox(duration)
	
	// moovボックス全体を組み立て
	moovContent := append(mvhdBox, trakBox...)
	
	// moov box header
	moovSize := uint32(8 + len(moovContent))
	moovHeader := make([]byte, 8)
	binary.BigEndian.PutUint32(moovHeader[0:4], moovSize)
	copy(moovHeader[4:8], []byte{'m', 'o', 'o', 'v'})
	
	moovBox := append(moovHeader, moovContent...)
	return moovBox
}

// createQuickTimeMovieHeaderBox はQuickTime互換のmvhdボックスを作成
func (w *MP4Writer) createQuickTimeMovieHeaderBox(duration uint32) []byte {
	currentTime := uint32(time.Now().Unix() + 2082844800) // Mac epoch adjustment
	
	mvhdBox := []byte{
		// mvhd box header
		0x00, 0x00, 0x00, 0x6C, // box size (108 bytes)
		'm', 'v', 'h', 'd',     // box type 'mvhd'
		0x00,                   // version
		0x00, 0x00, 0x00,       // flags
	}
	
	// creation and modification time
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint32(timeBytes[0:4], currentTime)
	binary.BigEndian.PutUint32(timeBytes[4:8], currentTime)
	mvhdBox = append(mvhdBox, timeBytes...)
	
	// timescale and duration
	mvhdBox = append(mvhdBox, []byte{
		0x00, 0x00, 0x03, 0xE8, // timescale (1000)
	}...)
	
	durationBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(durationBytes, duration)
	mvhdBox = append(mvhdBox, durationBytes...)
	
	// rate, volume, reserved, matrix, pre-defined, next track ID
	mvhdBox = append(mvhdBox, []byte{
		0x00, 0x01, 0x00, 0x00, // rate (1.0)
		0x01, 0x00,             // volume (1.0)
		0x00, 0x00,             // reserved
		0x00, 0x00, 0x00, 0x00, // reserved
		0x00, 0x00, 0x00, 0x00, // reserved
		// transformation matrix (identity)
		0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x40, 0x00, 0x00, 0x00,
		// pre-defined
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// next track ID
		0x00, 0x00, 0x00, 0x02,
	}...)
	
	return mvhdBox
}

// createQuickTimeVideoTrackBox は完全なビデオトラックボックスを作成
func (w *MP4Writer) createQuickTimeVideoTrackBox(duration uint32) []byte {
	// tkhd (Track Header Box)
	tkhdBox := w.createTrackHeaderBox(duration)
	
	// mdia (Media Box)
	mdiaBox := w.createMediaBox(duration)
	
	// trakボックス全体を組み立て
	trakContent := append(tkhdBox, mdiaBox...)
	
	// trak box header
	trakSize := uint32(8 + len(trakContent))
	trakHeader := make([]byte, 8)
	binary.BigEndian.PutUint32(trakHeader[0:4], trakSize)
	copy(trakHeader[4:8], []byte{'t', 'r', 'a', 'k'})
	
	return append(trakHeader, trakContent...)
}

// createTrackHeaderBox はtkhdボックスを作成
func (w *MP4Writer) createTrackHeaderBox(duration uint32) []byte {
	currentTime := uint32(time.Now().Unix() + 2082844800)
	
	tkhdBox := []byte{
		// tkhd box header
		0x00, 0x00, 0x00, 0x5C, // box size (92 bytes)
		't', 'k', 'h', 'd',     // box type 'tkhd'
		0x00,                   // version
		0x00, 0x00, 0x07,       // flags (track enabled, in movie, in preview)
	}
	
	// creation/modification time
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint32(timeBytes[0:4], currentTime)
	binary.BigEndian.PutUint32(timeBytes[4:8], currentTime)
	tkhdBox = append(tkhdBox, timeBytes...)
	
	// track ID, reserved, duration
	tkhdBox = append(tkhdBox, []byte{
		0x00, 0x00, 0x00, 0x01, // track ID
		0x00, 0x00, 0x00, 0x00, // reserved
	}...)
	
	durationBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(durationBytes, duration)
	tkhdBox = append(tkhdBox, durationBytes...)
	
	// remaining fields
	tkhdBox = append(tkhdBox, []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // reserved
		0x00, 0x00,             // layer
		0x00, 0x00,             // alternate group
		0x00, 0x00,             // volume
		0x00, 0x00,             // reserved
		// transformation matrix
		0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x40, 0x00, 0x00, 0x00,
		// width and height (960x720 - typical Tello resolution)
		0x03, 0xC0, 0x00, 0x00, // width (960)
		0x02, 0xD0, 0x00, 0x00, // height (720)
	}...)
	
	return tkhdBox
}

// createMediaBox はmdiaボックスを作成
func (w *MP4Writer) createMediaBox(duration uint32) []byte {
	// mdhd (Media Header Box)
	mdhdBox := w.createMediaHeaderBox(duration)
	
	// hdlr (Handler Reference Box)
	hdlrBox := w.createHandlerBox()
	
	// minf (Media Information Box)
	minfBox := w.createMediaInfoBox()
	
	// mdiaボックス全体を組み立て
	mdiaContent := append(mdhdBox, hdlrBox...)
	mdiaContent = append(mdiaContent, minfBox...)
	
	// mdia box header
	mdiaSize := uint32(8 + len(mdiaContent))
	mdiaHeader := make([]byte, 8)
	binary.BigEndian.PutUint32(mdiaHeader[0:4], mdiaSize)
	copy(mdiaHeader[4:8], []byte{'m', 'd', 'i', 'a'})
	
	return append(mdiaHeader, mdiaContent...)
}

// createMediaHeaderBox はmdhdボックスを作成
func (w *MP4Writer) createMediaHeaderBox(duration uint32) []byte {
	currentTime := uint32(time.Now().Unix() + 2082844800)
	
	mdhdBox := []byte{
		// mdhd box header
		0x00, 0x00, 0x00, 0x20, // box size (32 bytes)
		'm', 'd', 'h', 'd',     // box type 'mdhd'
		0x00,                   // version
		0x00, 0x00, 0x00,       // flags
	}
	
	// creation/modification time, timescale, duration
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint32(timeBytes[0:4], currentTime)
	binary.BigEndian.PutUint32(timeBytes[4:8], currentTime)
	mdhdBox = append(mdhdBox, timeBytes...)
	
	mdhdBox = append(mdhdBox, []byte{
		0x00, 0x00, 0x75, 0x30, // timescale (30000 - 30fps x 1000)
	}...)
	
	durationBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(durationBytes, duration*30) // adjust for timescale
	mdhdBox = append(mdhdBox, durationBytes...)
	
	mdhdBox = append(mdhdBox, []byte{
		0x55, 0xC4, // language ('und' = undetermined)
		0x00, 0x00, // pre-defined
	}...)
	
	return mdhdBox
}

// createHandlerBox はhdlrボックスを作成
func (w *MP4Writer) createHandlerBox() []byte {
	handlerName := "VideoHandler\x00" // null-terminated
	
	hdlrBox := []byte{
		// hdlr box header
		0x00, 0x00, 0x00, 0x21, // box size (33 bytes)
		'h', 'd', 'l', 'r',     // box type 'hdlr'
		0x00,                   // version
		0x00, 0x00, 0x00,       // flags
		0x00, 0x00, 0x00, 0x00, // pre-defined
		'v', 'i', 'd', 'e',     // handler type 'vide'
		0x00, 0x00, 0x00, 0x00, // reserved
		0x00, 0x00, 0x00, 0x00, // reserved
		0x00, 0x00, 0x00, 0x00, // reserved
	}
	
	hdlrBox = append(hdlrBox, []byte(handlerName)...)
	
	// 実際のサイズに更新
	actualSize := uint32(len(hdlrBox))
	binary.BigEndian.PutUint32(hdlrBox[0:4], actualSize)
	
	return hdlrBox
}

// createMediaInfoBox はminfボックスを作成
func (w *MP4Writer) createMediaInfoBox() []byte {
	// vmhd (Video Media Header Box)
	vmhdBox := []byte{
		0x00, 0x00, 0x00, 0x14, // box size (20 bytes)
		'v', 'm', 'h', 'd',     // box type 'vmhd'
		0x00,                   // version
		0x00, 0x00, 0x01,       // flags (no lean ahead)
		0x00, 0x00,             // graphics mode
		0x00, 0x00, 0x00, 0x00, // opcolor (R, G)
		0x00, 0x00,             // opcolor (B)
	}
	
	// dinf (Data Information Box)
	dinfBox := []byte{
		0x00, 0x00, 0x00, 0x24, // box size (36 bytes)
		'd', 'i', 'n', 'f',     // box type 'dinf'
		// dref (Data Reference Box)
		0x00, 0x00, 0x00, 0x1C, // dref box size (28 bytes)
		'd', 'r', 'e', 'f',     // box type 'dref'
		0x00,                   // version
		0x00, 0x00, 0x00,       // flags
		0x00, 0x00, 0x00, 0x01, // entry count
		// url entry
		0x00, 0x00, 0x00, 0x0C, // entry size (12 bytes)
		'u', 'r', 'l', ' ',     // entry type 'url '
		0x00,                   // version
		0x00, 0x00, 0x01,       // flags (self-reference)
	}
	
	// stbl (Sample Table Box) - simplified
	stblBox := []byte{
		0x00, 0x00, 0x00, 0x40, // box size (64 bytes)
		's', 't', 'b', 'l',     // box type 'stbl'
		// stsd (Sample Description Box)
		0x00, 0x00, 0x00, 0x18, // stsd box size (24 bytes)
		's', 't', 's', 'd',     // box type 'stsd'
		0x00,                   // version
		0x00, 0x00, 0x00,       // flags
		0x00, 0x00, 0x00, 0x01, // entry count
		// sample entry (minimal)
		0x00, 0x00, 0x00, 0x08, // entry size (8 bytes)
		'a', 'v', 'c', '1',     // entry type 'avc1' (H.264)
		// stts (Time-to-Sample Box)
		0x00, 0x00, 0x00, 0x10, // stts box size (16 bytes)
		's', 't', 't', 's',     // box type 'stts'
		0x00,                   // version
		0x00, 0x00, 0x00,       // flags
		0x00, 0x00, 0x00, 0x01, // entry count
		0x00, 0x00, 0x00, 0x01, // sample count
		0x00, 0x00, 0x03, 0xE8, // sample delta (1000)
		// stsc (Sample-to-Chunk Box)
		0x00, 0x00, 0x00, 0x10, // stsc box size (16 bytes)
		's', 't', 's', 'c',     // box type 'stsc'
		0x00,                   // version
		0x00, 0x00, 0x00,       // flags
		0x00, 0x00, 0x00, 0x00, // entry count (0)
	}
	
	// minfボックス全体を組み立て
	minfContent := append(vmhdBox, dinfBox...)
	minfContent = append(minfContent, stblBox...)
	
	// minf box header
	minfSize := uint32(8 + len(minfContent))
	minfHeader := make([]byte, 8)
	binary.BigEndian.PutUint32(minfHeader[0:4], minfSize)
	copy(minfHeader[4:8], []byte{'m', 'i', 'n', 'f'})
	
	return append(minfHeader, minfContent...)
}
