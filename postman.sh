#!/bin/bash

# 必要なディレクトリを作成
mkdir -p reports

# テストパラメータ
ITERATIONS=1000      # 繰り返し回数
CONCURRENCY=50      # 同時実行数
DELAY=0             # リクエスト間の遅延(ms)

# Docker Composeでアプリを起動
echo "アプリケーションを起動中..."
docker-compose up -d

# アプリケーションの準備を待機（10秒）
echo "アプリケーション起動を待機中..."
sleep 10

# Newmanで負荷テスト実行
echo "負荷テストを実行中..."
newman run stock-api-collection.json \
  -e environment.json \
  --iteration-count $ITERATIONS \
  --iteration-data data.json \
  --reporters cli,htmlextra,json \
  --reporter-htmlextra-export reports/report.html \
  --reporter-json-export reports/report.json \
  --insecure \
  --delay-request $DELAY \
  --timeout-request 10000 \
  --bail

# 終了コードを取得
TEST_EXIT_CODE=$?

# 結果の表示
if [ $TEST_EXIT_CODE -eq 0 ]; then
  echo "負荷テストが正常に完了しました"
else
  echo "負荷テストが失敗しました"
fi

# 統計情報を表示
if [ -f reports/report.json ]; then
  echo "----- テスト統計情報 -----"
  jq '.run.stats' reports/report.json
  echo "--------------------------"
fi

# テスト終了後、アプリケーションを停止（オプション）
# echo "アプリケーションを停止中..."
# docker-compose down

exit $TEST_EXIT_CODE
