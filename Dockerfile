FROM golang:1.26-alpine AS builder
WORKDIR /src
RUN apk add --no-cache upx
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o /out/kromgo ./cmd/kromgo
RUN upx --best --lzma /out/kromgo

FROM alpine AS fonts
RUN apk add --no-cache msttcorefonts-installer && update-ms-fonts

FROM gcr.io/distroless/static-debian12:nonroot
USER nonroot:nonroot
WORKDIR /kromgo
COPY --from=builder --chmod=555 /out/kromgo /kromgo/kromgo
COPY --from=fonts --chmod=555 /usr/share/fonts/truetype/msttcorefonts/Verdana.ttf /kromgo/
EXPOSE 8080/tcp 8888/tcp
LABEL \
    org.opencontainers.image.title="kromgo" \
    org.opencontainers.image.source="https://github.com/kashalls/kromgo"
ENTRYPOINT ["/kromgo/kromgo"]
