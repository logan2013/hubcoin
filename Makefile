.PHONY: all test get_deps

all: test install

NOVENDOR = go list github.com/dragisroua/hubcoin/... | grep -v /vendor/

build:
	go build github.com/dragosroua/hubcoin/cmd/...

install:
	go install github.com/dragosroua/hubcoin/cmd/...

test:
	go test --race `${NOVENDOR}`
	#go run tests/tendermint/*.go

get_deps:
	go get -d github.com/dragosroua/hubcoin/...

update_deps:
	go get -d -u github.com/dragosroua/hubcoin/...

get_vendor_deps:
	go get github.com/Masterminds/glide
	glide install

