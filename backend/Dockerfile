FROM golang:1.26 AS verifier
WORKDIR /src
COPY go.mod go.sum ./
COPY internal/domain ./internal/domain
COPY cmd/verifier ./cmd/verifier
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /rogue-verifier ./cmd/verifier

FROM python:3.12-slim

WORKDIR /srv

COPY backend/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY backend/app ./app
COPY --from=verifier /rogue-verifier /usr/local/bin/rogue-verifier

EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8000/api/health')"

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
