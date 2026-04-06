BINDIR := build
CMDS := $(notdir $(wildcard cmd/*))
LDFLAGS := -s -w

.PHONY: all build test clean docker-% $(CMDS)

all: build

build: $(CMDS)

$(CMDS):
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/$@ ./cmd/$@

test:
	go test ./...

clean:
	rm -rf $(BINDIR)

docker-%:
	docker build -f docker/$*.Dockerfile -t traffino/mcp-$*:latest .

docker-all: $(addprefix docker-,$(CMDS) memory)
