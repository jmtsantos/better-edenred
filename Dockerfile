FROM golang:1.17
COPY . /go/src/better-edenred/
WORKDIR /go/src/better-edenred/
RUN go install -ldflags="-s -w" ./...
ENTRYPOINT better-edenred