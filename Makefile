# Use ':=' instead of '=' to avoid multiple evaluation of NOW.
# # Substitute problematic characters with underscore using tr,
# #   make doesn't like spaces and ':' in filenames.
#NOW := $(shell date +"%c" | tr ' :' '__')
NOW := $(shell date +"%s" )

redskull:
	@echo building redskull binary
	@go vet
	@go build

dist-tar:  redskull
	@echo building distribution tarball
	@mkdir -p work/usr/redskull work/etc/supervisor.d
	@cp -a html work/usr/redskull/redskull
	@cp docker/supervisord/redskull.conf work/etc/supervisor.d/
	@cd work; tar -czf ../redskull-$(NOW).tar.gz *; cd ..
	@echo Your distribution tarball is redskull-$(NOW).tar.gz

docker-image: redskull
	@echo "Hope you have docker setup and have access ;)"
	docker build -t redskull .

docker-nolocalgo:
	@echo using centurylink/golang-builder to build docker container
	docker pull centurylink/golang-builder 
	docker run --rm -v ${PWD}:/src -v /var/run/docker.sock:/var/run/docker.sock  centurylink/golang-builder


.PHONY: clean
clean:
	@rm -f redskull
