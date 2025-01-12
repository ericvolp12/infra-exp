FROM golang:1.23.4 as builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd cmd

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /bin/pingcli ./cmd/ping

FROM ubuntu:24.04

COPY --from=builder /bin/pingcli /bin/pingcli

ENTRYPOINT ["/bin/pingcli"]
