FROM golang:1.26.8 AS verifier
WORKDIR /src
COPY go.mod go.sum ./
COPY internal/domain ./internal/domain
COPY cmd/verifier ./cmd/verifier
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /rogue-verifier ./cmd/verifier

FROM python:3.12.15-slim

WORKDIR /srv

COPY backend/requirements.txt .
# Точные версии с хешами (как их обновлять, написано в requirements.in):
# изменённый или неожиданный пакет ломает сборку.
RUN pip install --no-cache-dir --require-hashes -r requirements.txt

COPY backend/app ./app
COPY --from=verifier /rogue-verifier /usr/local/bin/rogue-verifier

# API и верификатору не нужен root: файлы остаются у root и только для чтения.
RUN useradd --system --uid 10001 --no-create-home --shell /usr/sbin/nologin app
USER app

EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8000/api/health')"

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
