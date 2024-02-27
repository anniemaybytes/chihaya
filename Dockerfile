FROM golang:1.22-alpine AS builder
WORKDIR /opt/chihaya

ARG CGO_ENABLED=0
ARG GOOS=linux
ARG GOARCH=amd64
ARG VERSION

RUN apk add binutils git make

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN make all

FROM scratch AS release
WORKDIR /app

COPY --from=builder /opt/chihaya/bin /

USER 1000:1000

CMD [ "/chihaya" ]
