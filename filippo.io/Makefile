.netlify/functions/go-get: go.mod go.sum netlify-functions/go-get/*.go
	go env
	go version 
	ls -l "$(GOPATH)/pkg/*"
	GOBIN=$(PWD)/.netlify/functions go install ./netlify-functions/go-get
