# syntax=docker/dockerfile:1
FROM golang:1.27.2-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/glauk ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/glauk /app/glauk
COPY --from=build /src/public /app/public
ENV PORT=3000
EXPOSE 3000
USER nonroot:nonroot
ENTRYPOINT ["/app/glauk"]
