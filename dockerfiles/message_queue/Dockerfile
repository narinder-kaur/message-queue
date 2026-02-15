FROM golang:1.26 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://proxy.golang.org
COPY . .
WORKDIR /src/cmd/message_queue
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o /out/message-queue ./

FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/message-queue /usr/local/bin/message-queue
ENTRYPOINT ["/usr/local/bin/message-queue"]
