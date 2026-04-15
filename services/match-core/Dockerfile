FROM golang:1.24-alpine AS build

WORKDIR /app

COPY go.mod ./go.mod
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY proto ./proto

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=mod -o /out/match-core ./cmd/match-core

FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app
ENV PORT=50052
ENV APP_ENV=production

COPY --from=build /out/match-core /app/match-core

USER nonroot:nonroot
EXPOSE 50052

ENTRYPOINT ["/app/match-core"]
