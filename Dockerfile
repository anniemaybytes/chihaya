FROM golang:1.14-alpine AS builder

WORKDIR /app

RUN apk add binutils

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN export CGO_ENABLED=0 && \
    export GOOS=linux && GOARCH=amd64 && \
    export VERSION=$(cat ./VERSION) && \
    export DATE=$(date -Iseconds) && \
    go build -v -x -installsuffix cgo -trimpath -ldflags="-X 'main.BuildDate=$DATE' -X 'main.BuildVersion=$VERSION'" -o chihaya && \
    strip chihaya

FROM scratch AS release

WORKDIR /app

COPY --from=builder /app/chihaya /chihaya

USER 1000:1000

CMD [ "/chihaya" ]
