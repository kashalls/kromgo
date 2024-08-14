FROM golang:1.22-alpine AS build
ARG PKG=github.com/kashalls/kromgo
ARG VERSION=dev
ARG REVISION=dev
WORKDIR /build
COPY . .
RUN go build -ldflags "-s -w -X main.Version=${VERSION} -X main.Gitsha=${REVISION}" ./cmd/kromgo


FROM alpine as fonts

RUN apk add --no-cache msttcorefonts-installer
RUN update-ms-fonts

FROM gcr.io/distroless/static-debian12:nonroot
USER nonroot:nonroot
COPY --from=build --chmod=555 /build/kromgo /kromgo/kromgo
COPY --from=fonts --chmod=555 /usr/share/fonts/truetype/msttcorefonts/Verdana.ttf /kromgo/
EXPOSE 8080/tcp 8888/tcp
WORKDIR /kromgo
LABEL \
    org.opencontainers.image.title="kromgo" \
    org.opencontainers.image.source="https://github.com/kashalls/kromgo"
ENTRYPOINT ["/kromgo/kromgo"]
