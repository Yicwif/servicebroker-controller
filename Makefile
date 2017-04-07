GOFILES:=$(shell find . -name '*.go' | grep -v -E '(./vendor)')

all: \
	bin/linux/servicebroker-controller \
	#bin/darwin/reboot-agent \
	#bin/darwin/reboot-controller

images: GVERSION=$(shell $(CURDIR)/git-version.sh)
images: bin/linux/servicebroker-controller
	docker build -f Dockerfile -t servicebroker-controller:$(GVERSION) .

check:
	@find . -name vendor -prune -o -name '*.go' -exec gofmt -s -d {} +
	@go vet $(shell go list ./... | grep -v '/vendor/')
	@go test -v $(shell go list ./... | grep -v '/vendor/')

vendor:
	dep ensure

clean:
	rm -rf bin

bin/%: LDFLAGS=-X main.Version=$(shell $(CURDIR)/git-version.sh)
bin/%: $(GOFILES)
	mkdir -p $(dir $@)
	GOOS=$(word 1, $(subst /, ,$*)) GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $@ 

