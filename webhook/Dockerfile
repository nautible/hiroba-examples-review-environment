FROM golang:1.19.5 as builder

WORKDIR /go/app

RUN apt-get -y update && apt-get install -y ca-certificates

COPY ./cmd/ ./cmd/
COPY ./go.mod  ./
COPY ./go.sum  ./

ARG CGO_ENABLED=0
ARG GOOS=linux
ARG GOARCH=amd64
RUN cd cmd && \
    go build \
    -o /go/app/bin/main

FROM scratch as runner

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/app/bin/main /app/main

EXPOSE 8080
ENTRYPOINT ["/app/main"]
