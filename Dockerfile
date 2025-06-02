FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown

# Install necessary build dependencies
RUN apk add --no-cache pkgconfig \
    gcc \
    opus-dev \
    libsamplerate \
    mpg123

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=1 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    go build -ldflags="-X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}'" -o bot github.com/makeitchaccha/text-to-speech

FROM alpine

RUN apk add --no-cache \
    ca-certificates \
    opus \
    libsamplerate \
    mpg123

COPY --from=build /build/bot /bin/bot

ENTRYPOINT ["/bin/bot"]

CMD ["-config", "/var/lib/config.toml"]
