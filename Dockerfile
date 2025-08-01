FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY pkg ./pkg
COPY config ./config
COPY internal ./internal
COPY main.go ./
COPY default.png ./

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o website-archiver

FROM archlinux:base

LABEL org.opencontainers.image.title="Website Archiver"
LABEL org.opencontainers.image.description="A Go-based tool for downloading web pages, snapshots from the Wayback Machine and creating into a ZIM file"
LABEL org.opencontainers.image.source="https://github.com/Sudo-Ivan/website-archiver"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.authors="Sudo-Ivan"

WORKDIR /app

RUN pacman -Syu --noconfirm && \
    pacman -S --noconfirm \
    imagemagick \
    zim-tools

COPY --from=builder /app/website-archiver /app/

RUN mkdir -p /app/archive

ENTRYPOINT ["/app/website-archiver"]

CMD ["--help"]