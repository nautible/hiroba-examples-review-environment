# project-gitlab-webhook

GitLabからのWebhookを受信してMergeRequestリソースの作成/削除を行う

## Webhookの設定

Webhookはプロジェクト＞Settings＞Webhooksから設定する

- URL
  - http://webhook-receiver.gitlab-webhook.svc.cluster.local/webhook?project=<プロジェクト名>
- Secret token
  - 任意（ただしwebhook-receiver側と合わせておく）
  - 現バージョンではプロジェクトごとに個別のシークレットにはしない
- Trigger
  - Merge request events

なお、クラウド上で実行する場合はexternal-secret-operatorを利用してシークレットを作成する。

## マージリクエストメッセージ

メッセージのサンプル

- [マージリクエスト 作成時](./examples/mergerequest_open.json)
- [マージリクエスト 承認時](./examples/mergerequest_approval.json)
- [マージリクエスト マージ時](./examples/mergerequest_complete.json)

### マージリクエストの判定方法

- 新規作成
  - object_kind : merge_request
  - object_attributes.state : opened
  - object_attributes.action : open
- 承認
  - object_kind : merge_request
  - object_attributes.state : opened
  - object_attributes.action : approved
- マージ実行
  - object_kind : merge_request
  - object_attributes.state : merged
  - object_attributes.action : merge
- ブランチ削除（マージリクエスト作成済みのブランチ）
  - object_kind : merge_request
  - object_attributes.state : closed
  - object_attributes.action : close

## webhook-receiverの導入

## 権限

gitlab-webhookネームスペース内にある以下のリソースの全操作権限のみ付与

- Group: review.nautible.com
- Resource: mergerequests

### ビルド

```bash
eval $(minikube docker-env)
docker build -t webhook-receiver:v0.0.1 .
```

### デプロイ

```bash
kubectl create ns gitlab-webhook
kubectl create secret generic webhook-credentials -n gitlab-webhook --from-literal=<プロジェクト名>=<シークレット>
kubectl apply -f manifests/
```

### Kubernetesからコンテナを削除

```bash
kubectl delete -f manifests/
```
