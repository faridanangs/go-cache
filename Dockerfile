FROM golang:1.22.1 as builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go get -d -v ./...

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .
RUN go install github.com/cosmtrek/air@latest

FROM alpine:latest

WORKDIR /app

COPY --from=builder /go/bin/air /usr/local/bin/air
COPY --from=builder /app/app /app/app

EXPOSE 8000
CMD ["./app", "-C", ".air.toml"]
