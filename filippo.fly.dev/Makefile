%.html: %.md
	go run ./md2html $<

.PHONY: all
all: $(patsubst %.md,%.html,$(wildcard *.md))
