#! /bin/sh
set -e
go mod tidy
GOOS=linux GOARCH=amd64 go build -o myquant ./cmd/backend/
echo 'build成功准备scp'`date`
scp ./myquant root@103.133.178.149:/root/apps/ws-quant/new
echo '推送成功'`date`
