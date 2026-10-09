BINARY_NAME = weakline
INSTALL_DIR = $(HOME)/.local/bin
LDFLAGS = -ldflags="-s -w"

all: build

test:
	go test ./...

bench:
	go test -bench=. -benchmem ./...

build:
	go build ${LDFLAGS} -o ${BINARY_NAME} .

install: all
	mkdir -p ${DESTDIR}${INSTALL_DIR}
	cp -f ${BINARY_NAME} ${DESTDIR}${INSTALL_DIR}
	chmod 755 ${DESTDIR}${INSTALL_DIR}/${BINARY_NAME}

uninstall:
	rm -f ${DESTDIR}${INSTALL_DIR}/${BINARY_NAME}

.PHONY: all test bench build clean install uninstall
