FROM golang:1.13 AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -tags "scrape" -o chihaya && \
    strip chihaya

FROM scratch AS release

WORKDIR /app

COPY --from=builder /app/chihaya /chihaya

CMD [ "/chihaya" ]
