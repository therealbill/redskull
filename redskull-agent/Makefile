# Use ':=' instead of '=' to avoid multiple evaluation of NOW.
# # Substitute problematic characters with underscore using tr,
# #   make doesn't like spaces and ':' in filenames.
#NOW := $(shell date +"%c" | tr ' :' '__')
NOW := $(shell date +"%s" )
UNAME := $(shell uname -s)
BUILD_DATE := `date +%Y-%m-%d\ %H:%M`
VERSIONFILE := version.go
BUILD_NUMBER_FILE := .buildno
BUILD_NUMBER := $(shell cat $(BUILD_NUMBER_FILE))

redskull:
	@echo Building native redskull binary
	@go vet
	@go build -ldflags "-X=main.BUILD $(NOW)"
ifeq ($(UNAME), Darwin)
	@echo Building Linux redskull binary
	@GOOS="linux" GOARCH="amd64" go build -o redskull-linux -ldflags "-X=main.BUILD $(NOW)"
endif

dist-tar:  redskull
	@echo building distribution tarball
	@mkdir -p work/usr/redskull work/etc/supervisor.d
	@cp -a html work/usr/redskull/redskull
	@cp docker/supervisord/redskull.conf work/etc/supervisor.d/
	@cd work; tar -czf ../redskull-$(NOW).tar.gz *; cd ..
	@echo Your distribution tarball is redskull-$(NOW).tar.gz

docker-image: redskull
	@echo "Hope you have docker setup and have access ;)"
	@GOOS="linux" GOARCH="amd64" go build -o redskull -ldflags "-X=main.BUILD $(NOW)"
	docker build -t redskull:$(NOW) .

docker-image-with-local-consul: redskull
	@echo "Hope you have docker setup and have access ;)"
	docker build -t redskull:consul:$(NOW) -f Dockerfile-consul .

docker-image-with-remote-consul: redskull
	@echo "Hope you have docker setup and have access ;)"
	docker build -t redskull:consul:$(NOW) -f Dockerfile-consul-remote .

# This is useful for when you have a local docker 
# # but no local Go installation
docker-nolocalgo:
	@echo using centurylink/golang-builder to build docker container
	docker pull centurylink/golang-builder 
	docker run --rm -v ${PWD}:/src -v /var/run/docker.sock:/var/run/docker.sock  centurylink/golang-builder

# for setting up a default config, useful mainly for test/dev and initialization
config-dev:
	curl -X PUT -d '0.0.0.0' http://localhost:8500/v1/kv/svcconfig/redskull-agent/cell0/config/bindaddress
	curl -X PUT -d '11000' http://localhost:8500/v1/kv/svcconfig/redskull-agent/cell0/config/rpcport
	curl -X PUT -d '1m' http://localhost:8500/v1/kv/svcconfig/redskull-agent/cell0/config/refreshinterval
	curl -X PUT -d '/etc/redis/sentinel.conf' http://localhost:8500/v1/kv/svcconfig/redskull-agent/cell0/config/sentinelconfig

