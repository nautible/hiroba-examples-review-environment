# hiroba-examples-review-environment

本リポジトリはオブジェクトの広場 [Kubernetesオペレーターパターンの活用](https://www.ogis-ri.co.jp/otc/hiroba/technical/kubernetes_use/part9.html) 確認用コードになります。※ オブジェクトの広場の記事は2023/3/28公開予定

## 動作環境

本サンプルは[hiroba-examples-project-create](https://github.com/nautible/hiroba-examples-project-create)で作成した環境を利用します。動作環境については[こちらのREADME](https://github.com/nautible/hiroba-examples-project-create)を参照してください。

## フォルダ構成

- manifests
  - Gatewayリソースデプロイ用マニフェスト
- operator
  - カスタムオペレーター実装コード
- webook
  - GitLabからWebhookを受け付けてカスタムリソースをデプロイするアプリケーション

## 実行手順

手順についてはオブジェクトの広場記事を参考にしてください。
