PROJECT := j2
SCRIPTDIR := $(shell pwd)
ROOTDIR := $(shell cd $(SCRIPTDIR) && pwd)
VERSION:= $(shell cat $(ROOTDIR)/VERSION)
COMMIT := $(shell git rev-parse --short HEAD)

GOBUILDDIR := $(SCRIPTDIR)/.gobuild
SRCDIR := $(SCRIPTDIR)
BINDIR := $(ROOTDIR)
VENDORDIR := $(ROOTDIR)/deps

ORGPATH := github.com/pulcy
ORGDIR := $(GOBUILDDIR)/src/$(ORGPATH)
REPONAME := $(PROJECT)
REPODIR := $(ORGDIR)/$(REPONAME)
REPOPATH := $(ORGPATH)/$(REPONAME)
BIN := $(BINDIR)/$(PROJECT)

GOPATH := $(GOBUILDDIR)
GOVERSION := 1.7beta1-alpine

ifndef GOOS
	GOOS := $(shell go env GOOS)
endif
ifndef GOARCH
	GOARCH := $(shell go env GOARCH)
endif

SOURCES := $(shell find $(SRCDIR) -name '*.go')

.PHONY: all clean deps

all: $(BIN)

clean:
	rm -Rf $(BIN) $(GOBUILDDIR)

deps:
	@${MAKE} -B -s $(GOBUILDDIR)

$(GOBUILDDIR):
	@mkdir -p $(ORGDIR)
	@rm -f $(REPODIR) && ln -s ../../../.. $(REPODIR)
	@GOPATH=$(GOPATH) pulsar go flatten -V $(VENDORDIR)

update-vendor:
	@rm -Rf $(VENDORDIR)
	@pulsar go vendor -V $(VENDORDIR) \
		github.com/cenkalti/backoff \
		github.com/coreos/etcd/client \
		github.com/coreos/fleet/client \
		github.com/dchest/uniuri \
		github.com/ewoutp/go-aggregate-error \
		github.com/gosuri/uilive \
		github.com/spf13/pflag \
		github.com/spf13/cobra \
		github.com/juju/errgo \
		github.com/mitchellh/mapstructure \
		github.com/hashicorp/go-rootcerts \
		github.com/hashicorp/hcl \
		github.com/hashicorp/vault/api \
		github.com/kr/pretty \
		github.com/kardianos/osext \
		github.com/mitchellh/go-homedir \
		github.com/nyarla/go-crypt \
		github.com/op/go-logging \
		github.com/pulcy/robin-api \
		github.com/ryanuber/columnize

$(BIN): $(GOBUILDDIR) $(SOURCES)
	docker run \
		--rm \
		-v $(ROOTDIR):/usr/code \
		-e GOPATH=/usr/code/.gobuild \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		-e CGO_ENABLED=0 \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		go build -a -installsuffix netgo -ldflags "-X main.projectVersion=$(VERSION) -X main.projectBuild=$(COMMIT)" -o /usr/code/$(PROJECT) $(REPOPATH)

run-tests:
	@make run-test test=$(REPOPATH)/jobs

update-tests:
	@make run-tests UPDATE-FIXTURES=1

run-test:
	@if test "$(test)" = "" ; then \
		echo "missing test parameter, that is, path to test folder e.g. './middleware/'."; \
		exit 1; \
	fi
	@docker run \
	    --rm \
	    -v $(shell pwd):/usr/code \
	    -e GOPATH=/usr/code/.gobuild \
		-e TEST_ENV=test-env \
		-e UPDATE-FIXTURES=$(UPDATE-FIXTURES) \
	    -w /usr/code \
		golang:$(GOVERSION) \
	    go test -v $(test)
