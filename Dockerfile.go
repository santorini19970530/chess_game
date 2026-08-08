# lean go backend image (serves UI + API + websocket; fairy-stockfish for AI play)
FROM debian:bookworm-slim AS fs-builder
RUN apt-get update \
  && apt-get install -y --no-install-recommends make g++ \
  && rm -rf /var/lib/apt/lists/*
WORKDIR /fs
COPY py_analyser/Fairy-Stockfish-fairy_sf_14/src/ ./
ARG TARGETARCH
RUN set -eux; \
  if [ "$TARGETARCH" = "arm64" ]; then ARCH=armv8; else ARCH=x86-64-modern; fi; \
  make -j"$(nproc)" build ARCH="$ARCH"; \
  mv stockfish /stockfish

FROM golang:1.22-bookworm AS go-builder
WORKDIR /app
COPY go_backend/ ./go_backend/
WORKDIR /app/go_backend
RUN CGO_ENABLED=0 go build -o /out/go_backend .

FROM debian:bookworm-slim
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/*
# layout matches compile-time paths used by frontendPath (runtime.Caller → /app/go_backend)
WORKDIR /app/go_backend
COPY --from=go-builder /out/go_backend /app/go_backend/go_backend
COPY frontend/ /app/frontend/
COPY --from=fs-builder /stockfish /usr/local/bin/stockfish
RUN chmod +x /usr/local/bin/stockfish /app/go_backend/go_backend \
  && touch /app/frontend/styles/style.css

ENV USE_FAIRY_STOCKFISH=true \
    FAIRY_STOCKFISH_PATH=/usr/local/bin/stockfish \
    PY_ANALYSER_URL=http://analyser:8001

EXPOSE 8080
CMD ["/app/go_backend/go_backend"]
