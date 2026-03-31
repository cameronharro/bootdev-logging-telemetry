APP_NAME := linko

all: build

build:
	go build \
		-ldflags "-X boot.dev/linko/internal/build.GitSHA=$$(git rev-parse HEAD) -X boot.dev/linko/internal/build.BuildTime=$$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
		-o $(APP_NAME) \
		.

dev: build
	rm -f ./linko.access.log
	LINKO_LOG_FILE=linko.access.log OTEL_EXPORTER_OTLP_TRACES_INSECURE=true ENV=development ./linko
