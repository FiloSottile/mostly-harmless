%.html: %.md md2html/*
	go run ./md2html $<

.PHONY: all
all: $(patsubst %.md,%.html,$(wildcard *.md))
