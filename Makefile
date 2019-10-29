# Makefile
# yuuki.miyo@gmail.com
# 2019/10/29

NAME     := go-twitter-dump-extractor
REPO     := github.com/yuukimiyo/go-twitter-dump-extractor
VERSION  := v0.0.1
REVISION := $(shell git rev-parse --short HEAD)

SRCS    := $(shell find . -type f -name '*.go')
LDFLAGS := -ldflags="-s -w -X \"$(REPO)/cmd.Version=$(VERSION)\" -X \"$(REPO)/cmd.Revision=$(REVISION)\" -extldflags \"-static\""

.PHONY: build
build: $(SRCS)
	go build -a -tags netgo -installsuffix netgo $(LDFLAGS) -o bin/$(NAME)

# >> make run ARGS="<args>"
.PHONY: run
run:
	go build -o bin/$(NAME)
	bin/$(NAME) $(ARGS)

.PHONY: dev
dev:
	go build -o bin/$(NAME)
	bin/$(NAME) -stderrthreshold=INFO -v=3 $(ARGS)

.PHONY: clean
clean:
	rm -rf bin/*
	# rm -rf vendor/*

.PHONY: doc
doc:
	godoc -http=:8080
