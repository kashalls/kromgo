FROM golang:1.23-alpine AS build
ARG PKG=github.com/kashalls/kromgo
ARG VERSION=dev
ARG REVISION=dev
WORKDIR /build
COPY . .
RUN go build -ldflags "-s -w -X main.Version=${VERSION} -X main.Gitsha=${REVISION}" ./cmd/kromgo

FROM gcr.io/distroless/static-debian12:nonroot
USER nonroot:nonroot
COPY --from=build --chmod=555 /build/kromgo /kromgo/kromgo
EXPOSE 8080/tcp 8888/tcp
LABEL \
    org.opencontainers.image.title="kromgo" \
    org.opencontainers.image.source="https://github.com/kashalls/kromgo"
ENTRYPOINT ["/kromgo/kromgo"]