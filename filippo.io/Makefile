.netlify/functions/go-get: go.mod go.sum netlify-functions/go-get/*.go
	find "$(GOPATH)/pkg" -type f | wc -l
	GOBIN=$(PWD)/.netlify/functions go install ./netlify-functions/go-get
	find "$(GOPATH)/pkg" -type f | wc -l
