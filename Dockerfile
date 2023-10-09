# Compile app binary
FROM golang:1.21.2 as build-env

WORKDIR /build

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go install -installsuffix cgo  -ldflags '-extldflags "-static"' ./cmd/...

# Run app in scratch
FROM scratch

COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-env /go/bin /app

ENV PATH="/app:${PATH}"

CMD ["mynews"]