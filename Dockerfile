# Build Project
FROM golang:1.22.4-alpine as build
WORKDIR /go/src/github.com/kashalls/kromgo

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT=""

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    GOARM=${TARGETVARIANT}

COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN go build -ldflags="-s -w" -o /kromgo

# Final Image
FROM gcr.io/distroless/static:nonroot
USER nonroot:nonroot
COPY --from=build --chown=nonroot:nonroot /kromgo /kromgo/
EXPOSE 8080

CMD ["/kromgo/kromgo"]
LABEL \
    org.opencontainers.image.title="kromgo" \
    org.opencontainers.image.source="https://github.com/kashalls/kromgo"