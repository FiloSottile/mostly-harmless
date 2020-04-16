.PHONY: vendor/bin/gopherjs bin/dnskey-to-ds.js

bin/dnskey-to-ds.js: vendor/bin/gopherjs
	GOPATH=${PWD}:${PWD}/vendor ./vendor/bin/gopherjs install dnskey-to-ds

vendor/bin/gopherjs:
	GOPATH=${PWD}:${PWD}/vendor go install github.com/gopherjs/gopherjs
