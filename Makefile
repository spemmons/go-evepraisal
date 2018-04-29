default: build

PKG_DIRS=$(shell go list ./... | grep -v /vendor/)
TEST_REPORT_PATH ?= target/reports
ENV?=dev
ifeq ($(ENV), dev)
	BUILD_OPTS?=-tags dev
	BINDATA_FLAGS?=-debug
else
	BUILD_OPTS?=
endif

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
GOPATH?=$(shell go env GOPATH)
export PATH := $(PATH):$(GOPATH)/bin

.PHONY: setup build install generate clean test test-reload run run-reload dist deploy

setup:
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
	go get -u github.com/jteeuwen/go-bindata/...
	go get -u github.com/cespare/reflex
	go get -u github.com/jstemmer/go-junit-report
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install
	dep ensure -dry-run -v

build: generate
	go build ${BUILD_OPTS} -o ./target/evepraisal-${GOOS}-${GOARCH} ./evepraisal

install: generate
	go install ${BUILD_OPTS} ${PKG_DIRS}

generate:
	go generate ${BUILD_OPTS} ${PKG_DIRS}

clean:
	go clean ./...
	rm -rf target

test:
	go vet ${PKG_DIRS}
	mkdir -p ${TEST_REPORT_PATH}
	go test ${PKG_DIRS} -v 2>&1 | tee ${TEST_REPORT_PATH}/test-output.txt
	cat ${TEST_REPORT_PATH}/test-output.txt | go-junit-report -set-exit-code > ${TEST_REPORT_PATH}/test-report.xml

test-reload:
	reflex -c reflex.test.conf

lint:
	gometalinter \
		--vendored-linters \
		--vendor \
		--disable-all \
		--line-length=180 \
		--cyclo-over=50 \
		--exclude="^web/bindata\.go:" \
		--enable=gocyclo \
		--enable=gas \
		--enable=goconst \
		--enable=gofmt \
		--enable=goimports \
		--enable=golint \
		--enable=gotype \
		--enable=ineffassign \
		--enable=lll \
		--enable=misspell \
		--enable=vet \
		--enable=vetshadow \
		./...

run: install
	evepraisal

run-reload:
	reflex -c reflex.conf

dist:
	ENV=PROD GOOS=linux GOARCH=amd64 make build

deploy-prod: dist
	USERNAME=root HOSTNAME=new.evepraisal.com ./scripts/deploy.sh
