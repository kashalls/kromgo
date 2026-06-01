# Vendor the npm packages (marked, github-markdown-css, @mdi/svg, simple-icons) that
# become the embedded assets. Pinned by package-lock.json and kept current by Renovate.
FROM node:24-alpine AS assets
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

FROM golang:1.26-alpine AS builder
ARG VERSION=dev
ARG REVISION=dev
WORKDIR /src
RUN apk add --no-cache upx ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# genassets turns the vendored node_modules into the //go:embed assets at build time;
# neither node_modules nor the generated assets are committed to the repo.
COPY --from=assets /src/node_modules ./node_modules
RUN go run ./cmd/genassets
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=${VERSION} -X main.Gitsha=${REVISION}" -trimpath -o /out/kromgo ./cmd/kromgo
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
