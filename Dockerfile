FROM golang:1.26.2-alpine AS builder

ARG APP=api
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags='-s -w' -o /out/app ./cmd/${APP}

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /out/app /app/app

EXPOSE 8080

ENTRYPOINT ["/app/app"]