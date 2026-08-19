FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod .
RUN go mod download

COPY *.go .
RUN go build -o /app/bin

FROM alpine:3.18

WORKDIR /app 

COPY --from=builder /app/bin /app/bin

CMD ["/app/bin"]

