FROM golang:1.24-alpine AS build

WORKDIR /app

COPY go.mod ./go.mod
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY proto ./proto

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=mod -o /out/bot-engine ./cmd/bot-engine

FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app
ENV PORT=50051
ENV APP_ENV=production

COPY --from=build /out/bot-engine /app/bot-engine

USER nonroot:nonroot
EXPOSE 50051

ENTRYPOINT ["/app/bot-engine"]
