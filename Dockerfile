FROM golang:1.14-alpine AS builder
WORKDIR /src

ARG CGO_ENABLED=0
ARG GOOS=linux
ARG GOARCH=amd64
ARG GIT_SHA=local

RUN apk add binutils

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN export VERSION=$(cat ./VERSION) && export DATE=$(date -Iseconds) && \
    echo $VERSION-$GIT_SHA@$DATE && mkdir /out && \
    go build -o /out -v -ldflags="-X 'main.GitSHA=$GIT_SHA' -X 'main.BuildDate=$DATE' -X 'main.BuildVersion=$VERSION'" ./cmd/... && \
    strip /out/*

FROM scratch AS release
WORKDIR /app

COPY --from=builder /out /

USER 1000:1000

CMD [ "/chihaya" ]
