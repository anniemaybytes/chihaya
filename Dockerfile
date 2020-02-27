FROM golang:1.13 AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN export CGO_ENABLED=0 && export VERSION=$(cat ./VERSION) && export DATE=$(date -Iseconds) && \
    go build -ldflags="-X 'main.BuildDate=$DATE' -X 'main.BuildVersion=$VERSION'" -tags "scrape" \
    -o chihaya && strip chihaya

FROM scratch AS release

WORKDIR /app

COPY --from=builder /app/chihaya /chihaya

CMD [ "/chihaya" ]
