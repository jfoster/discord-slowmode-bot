.DEFAULT_GOAL = build

BINARY=discord-set-slowmode-bot
VERSION=$(shell git describe --tags --always)
LDFLAGS=-ldflags "-X main.version=${VERSION}"

release=mkdir ${BINARY}-${VERSION}-$(1)-$(2); cp cfg.yaml.template ${BINARY}-${VERSION}-$(1)-$(2)/cfg.yaml.template; GOOS=$(1) GOARCH=$(2) go build ${LDFLAGS} -o ${BINARY}-${VERSION}-$(1)-$(2)/${BINARY}-${VERSION}-$(1)-$(2)$(3); zip -r ${BINARY}-${VERSION}-$(1)-$(2).zip ${BINARY}-${VERSION}-$(1)-$(2) -i '*$(1)*' 'cfg.yaml.template' 

test:
	go test -v

run:
	go run ${LDFLAGS} .

build:
	go build ${LDFLAGS} -o ${BINARY}

release:
	$(call release,darwin,amd64,)
	$(call release,linux,amd64,)
	$(call release,windows,amd64,.exe)
	
clean:
	go clean
