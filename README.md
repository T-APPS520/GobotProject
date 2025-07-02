# Tello ドローンコントローラー

このプログラムは、DJI Telloドローンをキーボードで制御できるGoアプリケーションです。

## 機能

- **ドローン制御**: キーボードでドローンの離陸、着陸、移動を制御

## ファイル構成

### メインプログラム
- `main.go` - メインプログラム（エントリーポイント）
- `drone_controller.go` - Telloドローンを制御するクラス
- `camera_viewer.go` - カメラ画像を処理・表示するクラス
- `keyboard_handler.go` - キーボード入力を処理するクラス

### テストファイル
- `main_test.go` - メインプログラムの統合テスト
- `keyboard_handler_test.go` - キーボードハンドラーの単体テスト
- `keyboard_handler_coverage_test.go` - キーボードハンドラーのカバレッジ強化テスト
- `camera_viewer_test.go` - カメラビューワーのテスト

### 設定・ビルドファイル
- `go.mod` - Go モジュール定義
- `go.sum` - 依存関係のチェックサム
- `.gitignore` - Git除外設定
- `README.md` - このドキュメント

### 生成ファイル（実行時作成）
- `tello_controller.exe` - ビルド済み実行ファイル（Windows）
- `coverage.out` - テストカバレッジレポート
- `coverage.html` - HTML形式のカバレッジレポート

## 使用方法

### 1. 実行前の準備

1. Telloドローンの電源を入れる
2. PCのWi-FiでTelloのネットワーク（TELLO-xxxxxx）に接続
3. プログラムを実行

### 2. プログラムの実行

```bash
go run .
# または
go build -o tello_controller.exe
.\tello_controller.exe
```

### 3. キーボード操作

| キー | 動作 |
|------|------|
| **W** | 前進 |
| **A** | 左移動 |
| **S** | 後退 |
| **D** | 右移動 |
| **Space** | 上昇 |
| **Z** | 降下 |
| **Escape** | 離陸/着陸の切り替え |
| **Q** | プログラム終了 |

## テスト

### テストの実行

```bash
# 全テストを実行
go test -v ./...

# カバレッジ付きでテストを実行
go test -cover -v ./...

# カバレッジレポートを生成
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### テストカバレッジ

現在のテストカバレッジは約42.6%です。主要な機能とエラーハンドリングをカバーしています：

- キーボード操作の全パターン
- ドローンコントローラーの全メソッド
- カメラビューワーの基本機能
- エラー回復とシャットダウン処理
- 連続操作と緊急着陸シナリオ

## 必要なパッケージ

### 依存関係のインストール

```bash
# 依存関係を一括インストール
go mod tidy

# または個別にインストール
go get gobot.io/x/gobot
go get gobot.io/x/gobot/platforms/dji/tello
go get github.com/nsf/termbox-go
```

### 主な依存関係

- **gobot.io/x/gobot**: ロボティクス・IoTフレームワーク
- **gobot.io/x/gobot/platforms/dji/tello**: DJI Telloドローン用ドライバー
- **github.com/nsf/termbox-go**: ターミナルベースのユーザーインターフェース

## 開発情報

### プロジェクト特徴

- **モジュラー設計**: 各機能が独立したクラスに分離
- **テスト駆動**: 42.6%のテストカバレッジ（主要機能を網羅）
- **エラーハンドリング**: 堅牢なエラー回復とグレースフルシャットダウン
- **CI/CD対応**: テスト可能な設計で本番環境に適用可能

### アーキテクチャ

```
main.go
├── DroneController    # ドローン制御ロジック
├── CameraViewer      # カメラ・表示処理
└── KeyboardHandler   # ユーザー入力処理
```

## 注意事項

1. **安全な場所での使用**: ドローンは必ず安全な場所で使用してください
2. **バッテリー残量**: ドローンのバッテリー残量を確認してから使用してください
3. **Wi-Fi接続**: TelloドローンのWi-Fiネットワークに接続されていることを確認してください
4. **室内使用推奨**: 初回使用時は室内での使用を推奨します

## トラブルシューティング

### ドローンに接続できない場合
- TelloのWi-Fiネットワークに正しく接続されているか確認
- ドローンが起動しているか確認
- 他のTelloコントローラーアプリが起動していないか確認

### キーボード操作が効かない場合
- プログラムのコンソール画面がアクティブになっているか確認
- termbox-goパッケージが正しくインストールされているか確認
