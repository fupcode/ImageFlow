FROM node:slim AS frontend-builder
WORKDIR /frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
ARG NEXT_PUBLIC_API_URL=""
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV BROWSERSLIST_IGNORE_OLD_DATA=true
RUN npm run build

FROM golang:1.23-alpine AS builder
ENV GO111MODULE=on
WORKDIR /app
RUN apk add --no-cache git gcc musl-dev vips-dev libheif-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o imageflow

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates vips libheif && \
    mkdir -p /app/data /app/static/images/original/landscape /app/static/images/original/portrait /app/static/images/landscape/webp /app/static/images/landscape/avif /app/static/images/landscape/thumb /app/static/images/portrait/webp /app/static/images/portrait/avif /app/static/images/portrait/thumb
COPY --from=builder /app/imageflow /app/
COPY --from=frontend-builder /frontend/out /app/frontend

ENV API_KEY=""
ENV LOCAL_STORAGE_PATH="/app/static/images"
ENV CUSTOM_DOMAIN=""
ENV MAX_UPLOAD_COUNT="20"
ENV IMAGE_QUALITY="80"
ENV WORKER_THREADS="4"
ENV SPEED="5"
ENV METADATA_SQLITE_PATH="/app/data/metadata.db"
ENV DEBUG_MODE="false"

EXPOSE 8686

CMD ["./imageflow"]
