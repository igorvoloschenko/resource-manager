FROM golang:alpine as builder
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
RUN mkdir -p /app
WORKDIR /app
COPY . .
RUN go build -o main .

FROM alpine:3
RUN mkdir -p /app
WORKDIR /app
COPY --from=builder /app/main /usr/bin/
ENTRYPOINT ["main"]