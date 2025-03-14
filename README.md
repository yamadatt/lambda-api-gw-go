## 在庫管理API

API-GWとLambdaの構成でAPIを構築する。

APIは在庫を管理し、売上が発生したら在庫を引き当てるもの。

## このリポジトリで目指すもの

お勉強用のリポジトリであり、途中で目指したいものを忘れそうなので以下にメモ。

### やろうと思ったきっかけ

ECSを使用して、コンテナでAPIを動かしている。

ECSは使用してないときでも起動させておかなければならないし、ECSを動かすためのALBなどが高額。また、負荷がかかるようであればタスクを複数起動するような制御をするような仕組みが必要となる。

そこで、API-GWとLambdaなら安価にできるのではないかと思い立ち、初期衝動ではじめた。

ただ、APIについて無知なので、勉強も兼ねてやっている。

### 目指すもの

というわけで、以下を目指す。

- APIはフレームワークを使う
  - golangのGinを使用する
- 単体テストを書いてCIをまわす
  - DIを意識してモックなどを使用する
- データベースと連携してのテスト
  - テスト実行時にコンテナでデータベースを動かす
- ローカルでAPI-GWとLabmdaを模してテストできるようにする
- LambdaはSAMを使用してデプロイする
- データベースとの結合試験を実施する


テスト実行時にコンテナでデータベースを起動してテスト。

```bash
go test -v ./... -run TestAPIIntegration
```

go test -v -tags=swagger_integration ./... 



## 参考

実際に使用した便利コマンド



mysql -u[ユーザー名] -p -h [IP or HOST_NAME] --port [ポート番号] [DB_NAME]

mysql -u your_db_user -p

sam build;sam local start-api --host 192.168.1.78