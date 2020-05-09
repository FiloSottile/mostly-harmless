package main

import (
	"html/template"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var templateHTML = template.Must(template.New("go-get.html").Parse(`
{{ $repo := or .GitRepo (printf "https://github.com/FiloSottile/%s" .Name) }}
<head>
    <meta name="go-import" content="filippo.io/{{ .Name }} git {{ $repo }}">
    <meta http-equiv="refresh" content="0;URL='{{ or .Redirect $repo }}'">
<body>
    Nothing to see here, move along...
`))

func handler(request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	var data struct {
		Name     string
		GitRepo  string
		Redirect string
	}
	switch strings.TrimPrefix(request.Path, "/.netlify/functions/go-get") {
	case "/age":
		data.Name = "age"
	case "/cpace":
		data.Name = "cpace"
		data.GitRepo = "https://github.com/FiloSottile/go-cpace-ristretto255"
		data.Redirect = "https://pkg.go.dev/filippo.io/cpace"
	case "/mkcert":
		data.Name = "mkcert"
	case "/yubikey-agent":
		data.Name = "yubikey-agent"
	default:
		return &events.APIGatewayProxyResponse{
			StatusCode: 404,
			Body:       "unknown package",
		}, nil
	}
	buf := &strings.Builder{}
	templateHTML.Execute(buf, data)
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
