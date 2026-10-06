# Sprawl — build and install. Distros: make && make install PREFIX=/usr DESTDIR=pkg
PREFIX  ?= /usr/local
DESTDIR ?=
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo 0.1.0)
GOFLAGS ?= -trimpath
NAME    := sprawl

.PHONY: all build test check install uninstall clean licenses

all: build

build:
	go build $(GOFLAGS) -ldflags "-X github.com/antoniowav/sprawl/internal/meta.Version=$(VERSION)" -o $(NAME) ./cmd/sprawl

test:
	go test ./...

check: test
	@test -z "$$(gofmt -l .)" || (gofmt -l .; exit 1)
	go vet ./...

licenses:
	dist/licenses.sh

install: build
	install -Dm755 $(NAME) $(DESTDIR)$(PREFIX)/bin/$(NAME)
	./$(NAME) -write-icons $(DESTDIR)$(PREFIX)/share/icons/hicolor
	install -Dm644 dist/$(NAME).desktop $(DESTDIR)$(PREFIX)/share/applications/$(NAME).desktop
	install -Dm644 dist/$(NAME).metainfo.xml $(DESTDIR)$(PREFIX)/share/metainfo/$(NAME).metainfo.xml
	install -Dm644 docs/$(NAME).6 $(DESTDIR)$(PREFIX)/share/man/man6/$(NAME).6
	install -Dm644 README.md $(DESTDIR)$(PREFIX)/share/doc/$(NAME)/README.md
	install -Dm644 LICENSE $(DESTDIR)$(PREFIX)/share/licenses/$(NAME)/LICENSE
	install -Dm644 dist/THIRD_PARTY_LICENSES.txt $(DESTDIR)$(PREFIX)/share/licenses/$(NAME)/THIRD_PARTY_LICENSES.txt

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/$(NAME) \
	  $(DESTDIR)$(PREFIX)/share/applications/$(NAME).desktop \
	  $(DESTDIR)$(PREFIX)/share/metainfo/$(NAME).metainfo.xml \
	  $(DESTDIR)$(PREFIX)/share/man/man6/$(NAME).6
	rm -f $(DESTDIR)$(PREFIX)/share/icons/hicolor/*/apps/$(NAME).png
	rm -rf $(DESTDIR)$(PREFIX)/share/doc/$(NAME) $(DESTDIR)$(PREFIX)/share/licenses/$(NAME)

clean:
	rm -f $(NAME)
