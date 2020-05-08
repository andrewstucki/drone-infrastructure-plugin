FROM golang:1.14.2-alpine3.11 as builder

RUN apk update && apk add --no-cache git ca-certificates upx binutils && update-ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w" -o /bin/app && strip /bin/app && upx -9 /bin/app

FROM alpine:3.11

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/app /app

ENTRYPOINT ["/app"]
