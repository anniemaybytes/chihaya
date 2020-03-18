FROM golang:1.14-alpine AS builder

WORKDIR /app

ARG CGO_ENABLED=0

ARG GOOS=linux

ARG GOARCH=amd64

RUN apk add binutils

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN export VERSION=$(cat ./VERSION) && export DATE=$(date -Iseconds) && \
    go build -o .bin/ -v -trimpath -ldflags="-X 'main.BuildDate=$DATE' -X 'main.BuildVersion=$VERSION'" ./cmd/... && \
    strip .bin/*

FROM scratch AS release

WORKDIR /app

COPY --from=builder /app/.bin /

USER 1000:1000

CMD [ "/chihaya" ]
