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
    musl-dev \
    opus-dev \
    libsamplerate \
    mpg123-dev

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=1 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    go build -ldflags="-X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}'" -o bot github.com/makeitchaccha/text-to-speech

# install goose v3 latest
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

FROM alpine

WORKDIR /app

RUN apk add --no-cache \
    ca-certificates \
    opus \
    libsamplerate \
    mpg123

COPY --from=build /build/bot /bin/bot
COPY --from=build /go/bin/goose /bin/goose
COPY --from=build /build/locales /app/locales
COPY --from=build /build/migrations /app/migrations

ENTRYPOINT ["/bin/bot"]

CMD ["-config", "/var/lib/config.toml"]
