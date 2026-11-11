package main

import (
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// playerSpec は映像表示用外部プレイヤーの起動仕様
type playerSpec struct {
	name string   // 実行ファイル名
	args []string // 標準入力からH.264ストリームを読ませるための引数
}

// playerCandidates は表示に使える外部プレイヤーの候補（優先順）
//
// Telloの映像はH.264エレメンタリストリームとして届くため、
// 標準入力をデコードして表示できるプレイヤーを使う
// （Gobot公式のTelloサンプルと同じ方式）。
var playerCandidates = []playerSpec{
	{
		name: "ffplay",
		args: []string{
			"-f", "h264", // 入力形式: H.264エレメンタリストリーム
			"-fflags", "nobuffer", // バッファリングしない（低遅延）
			"-flags", "low_delay",
			"-framedrop", // 遅れたフレームは捨てて遅延を防ぐ
			"-probesize", "32",
			"-analyzeduration", "0",
			"-i", "-",
			"-window_title", "Tello Camera",
		},
	},
	{
		name: "mpv",
		args: []string{
			"--profile=low-latency",
			"--demuxer-lavf-format=h264",
			"--no-cache",
			"--window-title=Tello Camera",
			"-",
		},
	},
	{
		name: "mplayer",
		args: []string{
			"-demuxer", "h264es",
			"-fps", "30",
			"-cache", "128",
			"-title", "Tello Camera",
			"-",
		},
	},
}

// detectPlayer は利用可能な外部プレイヤーを検出する
func detectPlayer() (playerSpec, bool) {
	for _, candidate := range playerCandidates {
		if _, err := exec.LookPath(candidate.name); err == nil {
			return candidate, true
		}
	}
	return playerSpec{}, false
}

// VideoPlayer はH.264ストリームを外部プレイヤーへパイプしてPC画面に表示するクラス
type VideoPlayer struct {
	spec   playerSpec
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	exited chan struct{}
	mu     sync.Mutex
}

// NewVideoPlayer は利用可能なプレイヤーを自動検出して作成する
func NewVideoPlayer() (*VideoPlayer, error) {
	spec, ok := detectPlayer()
	if !ok {
		return nil, fmt.Errorf("映像表示用プレイヤー（ffplay / mpv / mplayer）が見つかりません。ffmpeg（ffplay）などをインストールしてください")
	}
	return newVideoPlayer(spec), nil
}

// newVideoPlayer は指定した仕様でプレイヤーを作成する（テスト用）
func newVideoPlayer(spec playerSpec) *VideoPlayer {
	return &VideoPlayer{spec: spec}
}

// Name はプレイヤー名を返す
func (p *VideoPlayer) Name() string {
	return p.spec.name
}

// Start はプレイヤープロセスを起動する
func (p *VideoPlayer) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd != nil {
		return nil // 起動済み
	}

	cmd := exec.Command(p.spec.name, p.spec.args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("プレイヤーの標準入力パイプ作成に失敗: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("プレイヤー（%s）の起動に失敗: %w", p.spec.name, err)
	}

	p.cmd = cmd
	p.stdin = stdin
	p.exited = make(chan struct{})

	// プロセス終了を監視してゾンビプロセス化を防ぐ
	go func() {
		cmd.Wait()
		close(p.exited)
	}()

	return nil
}

// Running はプレイヤーが起動中かどうかを返す
func (p *VideoPlayer) Running() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd == nil || p.exited == nil {
		return false
	}
	select {
	case <-p.exited:
		return false
	default:
		return true
	}
}

// Write はH.264フレームデータをプレイヤーへ送る（io.Writer）
func (p *VideoPlayer) Write(frameData []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stdin == nil {
		return 0, fmt.Errorf("プレイヤーが起動していません")
	}
	return p.stdin.Write(frameData)
}

// Close はプレイヤーを終了する
func (p *VideoPlayer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd == nil {
		return nil
	}

	if p.stdin != nil {
		p.stdin.Close() // EOFでプレイヤーに終了を促す
		p.stdin = nil
	}
	if p.cmd.Process != nil {
		p.cmd.Process.Kill() // 残っていれば確実に終了させる
	}
	p.cmd = nil
	return nil
}
