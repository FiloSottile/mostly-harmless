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
    Redirecting you to the <a href="{{ or .Redirect $repo }}">project page</a>...
`))

func handler(request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	var data struct {
		Name     string
		GitRepo  string
		Redirect string
	}
	// Note that when a "200 redirect" is followed, request.Path is the original one.
	path := strings.TrimPrefix(request.Path, "/.netlify/functions/go-get")
	pkg := func(n string) bool { return path == "/"+n || strings.HasPrefix(path, "/"+n+"/") }
	switch {
	case pkg("age"):
		data.Name = "age"
	case pkg("edwards25519"):
		data.Name = "edwards25519"
		data.Redirect = "https://pkg.go.dev/filippo.io/edwards25519"
	case pkg("cpace"):
		data.Name = "cpace"
		data.GitRepo = "https://github.com/FiloSottile/go-cpace-ristretto255"
		data.Redirect = "https://pkg.go.dev/filippo.io/cpace"
	case pkg("mkcert"):
		data.Name = "mkcert"
	case pkg("yubikey-agent"):
		data.Name = "yubikey-agent"
	case pkg("mostly-harmless"):
		data.Name = "mostly-harmless"
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
