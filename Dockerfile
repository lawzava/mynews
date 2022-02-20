# Compile app binary
FROM golang:1.17.7-alpine3.15 as build-env

WORKDIR /build

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install -installsuffix cgo  -ldflags '-extldflags "-static"' ./cmd/...

# Run app in scratch
FROM scratch

COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-env /go/bin /app

ENV PATH="/app:${PATH}"

CMD ["mynews"]