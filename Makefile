BINARY_NAME = weakline
INSTALL_DIR = $(HOME)/.local/bin
BUILD_FLAGS = -trimpath -ldflags="-s -w"

all: build

test:
	go test ./...

bench:
	go test -bench=. -benchmem ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

check: test bench vet

build:
	CGO_ENABLED=0 go build ${BUILD_FLAGS} -o ${BINARY_NAME} .

clean:
	rm -f ${BINARY_NAME}

install: all
	mkdir -p ${DESTDIR}${INSTALL_DIR}
	cp -f ${BINARY_NAME} ${DESTDIR}${INSTALL_DIR}
	chmod 755 ${DESTDIR}${INSTALL_DIR}/${BINARY_NAME}

uninstall:
	rm -f ${DESTDIR}${INSTALL_DIR}/${BINARY_NAME}

.PHONY: all test bench vet fmt check build clean install uninstall
