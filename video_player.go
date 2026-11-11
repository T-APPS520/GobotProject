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
// 標準入力へ H.264 をパイプして表示する方式
// 参考: https://github.com/hybridgroup/gophercon-2019/blob/HEAD/drone/tello/README.md
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
			"--title=Tello Camera", // mpvのウィンドウタイトル指定は --title（--window-title は ffplay 用）
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

// VideoPlayer は H.264 ストリームを外部プレイヤーへパイプして表示する
// mu はパイプI/Oの間は保持しない。writeMu は Write 同士の直列化専用
type VideoPlayer struct {
	spec    playerSpec
	mu      sync.Mutex
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	exited  chan struct{}
	writeMu sync.Mutex
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

	// 終了通知チャネルはこのプロセス専用として捕捉してから監視する
	exited := make(chan struct{})
	p.cmd = cmd
	p.stdin = stdin
	p.exited = exited

	// プロセス終了を監視してゾンビプロセス化を防ぐ
	go func() {
		cmd.Wait()
		close(exited)
	}()

	return nil
}

// Running はプレイヤーが起動中かどうかを返す
func (p *VideoPlayer) Running() bool {
	p.mu.Lock()
	exited := p.exited
	p.mu.Unlock()

	if exited == nil {
		return false
	}
	select {
	case <-exited:
		return false
	default:
		return true
	}
}

// Write は H.264 フレームデータをプレイヤーへ送る（io.Writer）
func (p *VideoPlayer) Write(frameData []byte) (int, error) {
	p.mu.Lock()
	stdin := p.stdin
	p.mu.Unlock()

	if stdin == nil {
		return 0, fmt.Errorf("プレイヤーが起動していません")
	}

	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	return stdin.Write(frameData)
}

// Close はプレイヤーを終了する
//
// 状態を取り出してクリアしてからロック外で終了処理を行う。
// Write がブロックしている最中でも即座に実行できる。
func (p *VideoPlayer) Close() error {
	p.mu.Lock()
	stdin := p.stdin
	cmd := p.cmd
	p.stdin = nil
	p.cmd = nil
	p.exited = nil
	p.mu.Unlock()

	if cmd == nil {
		return nil
	}

	if stdin != nil {
		stdin.Close() // EOFでプレイヤーに終了を促す
	}
	if cmd.Process != nil {
		cmd.Process.Kill() // 残っていれば確実に終了させる
	}
	return nil
}
