# --------------------
# Build 阶段
# --------------------
FROM golang:1.21.5-alpine3.19 AS builder

WORKDIR /go/src/

# 复制源码和配置文件
COPY ./ /go/src/

# 构建可执行文件
RUN go mod tidy && \
    go build -o netflowFlasher && \
    chmod +x ./netflowFlasher

# --------------------
# 运行阶段
# --------------------
FROM alpine

LABEL netflowflasher.image.author="Luckykeeper <https://luckykeeper.site>"
LABEL maintainer="Luckykeeper <https://luckykeeper.site>"

WORKDIR /app

# 复制可执行文件和 config.json
COPY --from=builder /go/src/netflowFlasher /app/netflowFlasher
COPY --from=builder /go/src/config.json /app/config.json

# 设置时区并清理 tzdata
RUN set -eux && \
    sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories && \
    apk --update add tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apk del tzdata

# 设置启动命令
ENTRYPOINT ["/app/netflowFlasher"]
