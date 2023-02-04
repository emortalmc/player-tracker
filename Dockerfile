FROM golang:alpine as go
WORKDIR /app
ENV GO111MODULE=on

COPY go.mod .
RUN go mod download

COPY . .
RUN go build -o player-tracker ./cmd

FROM alpine

WORKDIR /app

COPY --from=go /app/player-tracker ./player-tracker
COPY run/config.yaml ./config.yaml
CMD ["./player-tracker"]