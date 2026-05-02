PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
DATADIR ?= $(PREFIX)/share/tmux-agent

GO ?= go
INSTALL ?= install

BIN_DIR := bin
BIN := $(BIN_DIR)/tmux-agent
PLUGIN := agent-sidebar.tmux

.PHONY: all build install clean

all: build

build:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN) ./cmd/tmux-agent

install: build
	$(INSTALL) -d $(DESTDIR)$(BINDIR)
	$(INSTALL) -m 0755 $(BIN) $(DESTDIR)$(BINDIR)/tmux-agent
	$(INSTALL) -d $(DESTDIR)$(DATADIR)
	$(INSTALL) -m 0644 $(PLUGIN) $(DESTDIR)$(DATADIR)/$(PLUGIN)

clean:
	rm -rf $(BIN_DIR)
