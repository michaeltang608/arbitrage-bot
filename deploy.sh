#! /bin/sh
set -e
go mod tidy
GOOS=linux GOARCH=amd64 go build -o myquant ./cmd/backend/
echo 'build成功准备scp'
scp ./myquant root@104.194.239.171:/app/go-projects/ws_demo/new
echo '推送成功'`date`
