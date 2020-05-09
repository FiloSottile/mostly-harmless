.netlify/functions/go-get: go.mod go.sum netlify-functions/go-get/*.go
	find "$GOPATH/pkg"
	find "$GOCACHE"
	GOBIN=$(PWD)/.netlify/functions go install ./netlify-functions/go-get
