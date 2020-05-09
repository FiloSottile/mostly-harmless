.netlify/functions/go-get: go.mod go.sum netlify-functions/go-get/*.go
	go env
	ls -l "$(GOPATH)/pkg/mod"
	GOBIN=$(PWD)/.netlify/functions go install ./netlify-functions/go-get
