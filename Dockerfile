FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY netflowFlasher.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/netflowFlasher ./netflowFlasher.go

FROM alpine:3.21
WORKDIR /app
COPY --from=builder /out/netflowFlasher /app/netflowFlasher
COPY config.json /app/config.json
ENTRYPOINT ["/app/netflowFlasher"]
