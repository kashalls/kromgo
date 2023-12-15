FROM golang:1.21.5-alpine as build
WORKDIR /go/src/github.com/kashalls/kromgo

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/engine/reference/builder/#copy
COPY *.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags=”-s -w” -o /kromgo


FROM scratch
COPY —from=build /kromgo /kromgo/

EXPOSE 8080

# Run
CMD ["/kromgo/kromgo"]