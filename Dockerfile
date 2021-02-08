FROM golang:buster AS build-env

LABEL maintainer="Max Schmitt <max@schmitt.mx>"
LABEL description="FRITZ!Box Prometheus exporter"

COPY go.mod /go.mod
COPY go.sum /go.sum
RUN go mod download

COPY . /fritzbox_exporter

WORKDIR /fritzbox_exporter

RUN CGO_ENABLED=0 go build -o /out cmd/exporter/exporter.go

FROM alpine

RUN apk update && apk add ca-certificates

COPY --from=build-env /out /exporter

EXPOSE 9133

ENTRYPOINT ["/exporter"]
