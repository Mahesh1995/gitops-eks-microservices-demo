# Multi-stage build → tiny, non-root, static image.
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
# No external deps, so no `go mod download` step is required.
COPY . .
# CGO disabled → fully static binary that runs on a distroless/scratch base.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/frontend .

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/frontend /app/frontend
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/frontend"]
