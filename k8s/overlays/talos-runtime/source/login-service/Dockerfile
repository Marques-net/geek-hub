FROM golang:1.24-alpine AS build
WORKDIR /src
COPY . .
RUN go mod download && CGO_ENABLED=0 go build -o /out/login-service ./cmd/login-service

FROM alpine:3.20
RUN addgroup -S app && adduser -S app -G app
USER app
WORKDIR /app
COPY --from=build /out/login-service /app/login-service
EXPOSE 3000
ENTRYPOINT ["/app/login-service"]
