FROM golang:1.26 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://proxy.golang.org
COPY . .
WORKDIR /src/cmd/metrics
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o /out/metrics ./

FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/metrics /usr/local/bin/metrics
ENTRYPOINT ["/usr/local/bin/metrics"]
