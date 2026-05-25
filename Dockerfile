FROM node:20-alpine AS ui-builder

WORKDIR /src

COPY web/package*.json ./web/
RUN cd web && npm ci

COPY web ./web
COPY internal/ui ./internal/ui
RUN cd web && npm run build

FROM golang:1.26.1-alpine AS go-builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=ui-builder /src/internal/ui/static ./internal/ui/static

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/trove ./cmd/trove

FROM alpine:3.22

RUN addgroup -S trove \
    && adduser -S -G trove trove \
    && apk add --no-cache ca-certificates

COPY --from=go-builder /out/trove /trove

USER trove

EXPOSE 8080

ENTRYPOINT ["/trove"]
CMD ["serve"]
