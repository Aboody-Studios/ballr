# Stage 1: Build Go binary
FROM golang:1.26-alpine AS go-build

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./src/cmd/api/main.go

# Stage 2: Install Python dependencies
FROM python:3.12-slim AS python-deps

RUN apt-get update && apt-get install -y --no-install-recommends \
    libgl1-mesa-glx \
    libglib2.0-0 \
    libsm6 \
    libxext6 \
    libxrender-dev \
    libgomp1 \
    && rm -rf /var/lib/apt/lists/*

COPY src/cv/requirements.txt /tmp/requirements.txt
RUN pip install --no-cache-dir --extra-index-url https://download.pytorch.org/whl/cpu \
    -r /tmp/requirements.txt

# Stage 3: Production image
FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    libgl1-mesa-glx \
    libglib2.0-0 \
    libsm6 \
    libxext6 \
    libxrender-dev \
    libgomp1 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=go-build /app/server /app/server
COPY --from=python-deps /usr/local/lib/python3.12/site-packages/ /usr/local/lib/python3.12/site-packages/
COPY src/cv/ /app/cv/

ENV CV_SCRIPT_PATH=/app/cv/main.py
ENV CV_FPS_TARGET=10

WORKDIR /app
EXPOSE 3000
CMD ["/app/server"]
