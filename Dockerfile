FROM golang:1.26-alpine AS builder
WORKDIR /src
RUN apk add --no-cache upx ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Fetch the embedded assets (gallery marked.js/CSS + full MDI set) at build time —
# they are not committed to the repo; //go:embed bakes them into the binary.
RUN go run ./cmd/genassets
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o /out/kromgo ./cmd/kromgo
RUN upx --best --lzma /out/kromgo

FROM scratch
# kromgo dials Prometheus, which is commonly HTTPS, so the CA bundle is required.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/kromgo /kromgo
EXPOSE 8080/tcp 8888/tcp
LABEL \
    org.opencontainers.image.title="kromgo" \
    org.opencontainers.image.source="https://github.com/home-operations/kromgo"
# Run as whatever UID you configure (k8s securityContext / docker --user); the
# image pins no user.
ENTRYPOINT ["/kromgo"]
