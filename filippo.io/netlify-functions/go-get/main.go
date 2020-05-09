package main

import (
	"go/build"
	"html/template"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var templateHTML = template.Must(template.New("go-get.html").Parse(`
<body>
    <p>GOPATH is {{.GOPATH}} 
    <p>path is {{.PATH}} 
`))

func handler(request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	path := strings.TrimPrefix(request.Path, "/.netlify/functions/go-get")
	buf := &strings.Builder{}
	templateHTML.Execute(buf, map[string]string{
		"GOPATH": build.Default.GOPATH,
		"PATH":   path,
	})
	return &events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "text/html; charset=UTF-8",
		},
		Body: buf.String(),
	}, nil
}

func main() {
	lambda.Start(handler)
}
