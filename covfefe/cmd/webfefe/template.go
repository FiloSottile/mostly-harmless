package main

import (
	"html/template"
)

var htmlHeader = `
<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">

    <title>Covfefe</title>

    <link href="https://fonts.googleapis.com/css?family=Bitter|Raleway" rel="stylesheet">

    <style>
        .container {
            width: auto;
            max-width: 700px;
            padding: 0 15px;
        }

        body {
            font-family: "Raleway";
        }

        h1,
        h2 {
            font-family: "Bitter";
		}
		
		code {
            -webkit-font-smoothing: antialiased;
        }

        input {
            padding: 8px;
            font-size: 15px;
            font-family: "Raleway";
            min-width: 40%;
            height: 14px;
        }

        button {
            font-size: 20px;
            vertical-align: bottom;
            background: none;
            border: 1px solid lightgrey;
            border-radius: 5px;
            font-family: "Bitter";
            padding: 3px 6px;
        }
    </style>
</head>

<body>
    <div class="container">
        <h1>Covfefe</h1>

`

var tmplHome = template.Must(template.New("Home").Parse(htmlHeader + `
<p>There are {{.}} entries in the database.
`))

var tmplTweet = template.Must(template.New("TweetPage").Parse(htmlHeader + `
{{define "Tweet"}}
	<p>{{.User.Name}} <a href="https://twitter.com/{{.User.ScreenName}}">@{{.User.ScreenName}}</a>
	<p><blockquote>{{with .ExtendedTweet}}{{.FullText}}{{else}}{{.Text}}{{end}}</blockquote>
	<p>{{.CreatedAt}}
	<p>{{with printf "https://twitter.com/%s/status/%d" .User.ScreenName .ID}}
		<a href="{{.}}">{{.}}</a>
	{{end}}
{{end}}

<h2>Tweet number {{.ID}}</h2>
{{template "Tweet" .}}

{{with .RetweetedStatus}}
	<h3>Retweet of {{.ID}}</h3>
	{{template "Tweet" .}}

	{{with .QuotedStatus}}
		<h4>Quote of {{.ID}}</h4>
		{{template "Tweet" .}}
	{{end}}
{{end}}

{{with .QuotedStatus}}
	<h3>Quote of {{.ID}}</h3>
	{{template "Tweet" .}}
{{end}}
`))
