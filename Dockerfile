FROM golang:1.26-alpine AS builder
WORKDIR /src
RUN apk add --no-cache upx
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o /out/kromgo ./cmd/kromgo
RUN upx --best --lzma /out/kromgo

FROM gcr.io/distroless/static-debian12:nonroot
USER nonroot:nonroot
WORKDIR /kromgo
COPY --from=builder --chmod=555 /out/kromgo /kromgo/kromgo
EXPOSE 8080/tcp 8888/tcp
LABEL \
    org.opencontainers.image.title="kromgo" \
    org.opencontainers.image.source="https://github.com/home-operations/kromgo"
ENTRYPOINT ["/kromgo/kromgo"]
