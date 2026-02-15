FROM golang:1.26 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://proxy.golang.org
COPY . .
WORKDIR /src/cmd/consumer
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o /out/consumer ./

FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/consumer /usr/local/bin/consumer
ENTRYPOINT ["/usr/local/bin/consumer"]
