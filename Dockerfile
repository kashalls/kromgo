FROM golang:1.26-alpine AS builder
WORKDIR /src
RUN apk add --no-cache upx ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o /out/kromgo ./cmd/kromgo
RUN upx --best --lzma /out/kromgo

FROM scratch
# kromgo dials Prometheus, which is commonly HTTPS, so the CA bundle is required.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder --chmod=555 /out/kromgo /kromgo/kromgo
# Run unprivileged: a numeric UID needs no /etc/passwd entry (matches distroless nonroot).
USER 65532:65532
WORKDIR /kromgo
EXPOSE 8080/tcp 8888/tcp
LABEL \
    org.opencontainers.image.title="kromgo" \
    org.opencontainers.image.source="https://github.com/home-operations/kromgo"
ENTRYPOINT ["/kromgo/kromgo"]
