FROM golang:1.26 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://proxy.golang.org
COPY . .
WORKDIR /src/cmd/consumer
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o /out/consumer ./

FROM busybox:1.37.0-uclibc AS busybox

FROM gcr.io/distroless/static-debian12

# Now copy the static shell into base image.
COPY --from=busybox /bin/sh /bin/sh
COPY --from=busybox /bin/nc /bin/nc
COPY --from=busybox /bin/cut /bin/cut

COPY --from=builder /out/consumer /usr/local/bin/consumer
ENTRYPOINT ["/usr/local/bin/consumer"]
