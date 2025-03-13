#!/bin/bash

# 実行環境を指定（local, aws-sam）
ENV=${1:-local}
ENV_FILE="${ENV}-env.json"

# 繰り返し回数を指定
ITERATIONS=${2:-1000}

# 並列接続数
CONCURRENCY=${3:-50}

# 結果レポートのパス
REPORT_DIR="./reports/load-test"
TIMESTAMP=$(date +%Y%m%d%H%M%S)
REPORT_PATH="${REPORT_DIR}/load-test-${ENV}-${TIMESTAMP}.html"

echo "環境: $ENV, ファイル: $ENV_FILE"
echo "繰り返し回数: $ITERATIONS, 並列数: $CONCURRENCY"
echo "レポート: $REPORT_PATH"

# レポートディレクトリ作成
mkdir -p ${REPORT_DIR}

# Newmanをインストール（存在しない場合）
if ! command -v newman &> /dev/null; then
    echo "Newmanをインストールしています..."
    npm install -g newman newman-reporter-htmlextra
fi

# 負荷テストを実行
newman run stock-api-tests.json \
    -e ${ENV_FILE} \
    --folder "負荷テストシナリオ" \
    --iteration-count ${ITERATIONS} \
    --insecure \
    --reporters cli,htmlextra \
    --reporter-htmlextra-export ${REPORT_PATH}

echo "テスト完了: レポートは ${REPORT_PATH} に保存されました"
