package main

import "html/template"

var indexHTML = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <title>archive.today bundle - {{.Timestamp}}</title>
</head>

<body>

    <h1>archive.today bundle - {{.Timestamp}}</h1>
    <ul>
    {{range .Sites}}
        <li><a href="{{.Id}}/index.html">{{.Url}}</a></li>
    {{end}}
    </ul>

</body>
</html>
`))
