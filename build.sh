CGO_ENABLED=0 go build -ldflags="-w -s" && upx -9 ./aliddns