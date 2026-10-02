include config.mk

all: build

test:
	go test ./...

bench:
	go test -bench=. -benchmem ./...

build:
	mkdir -p ${BUILD_DIR}
	go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} .

clean:
	rm -rf ${BUILD_DIR}

install: all
	mkdir -p ${DESTDIR}${INSTALL_DIR}
	cp -f ${BUILD_DIR}/${BINARY_NAME} ${DESTDIR}${INSTALL_DIR}
	chmod 755 ${DESTDIR}${INSTALL_DIR}/${BINARY_NAME}

uninstall:
	rm -f ${DESTDIR}${INSTALL_DIR}/${BINARY_NAME}

.PHONY: all test bench build clean install uninstall
