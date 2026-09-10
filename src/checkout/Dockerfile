# Multi-stage build → tiny, non-root, static image.
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/checkout .

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/checkout /app/checkout
EXPOSE 4000
USER nonroot:nonroot
ENTRYPOINT ["/app/checkout"]
