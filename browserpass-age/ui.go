package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"text/template"

	"filippo.io/age/plugin"
)

var pluginHeadlessUI = &plugin.ClientUI{
	DisplayMessage: func(name, message string) error {
		return showNotification(message, fmt.Sprintf("age-plugin-%s", name))
	},
	RequestValue: prompt,
	Confirm:      confirm,
	WaitTimer: func(name string) {
		showNotification("waiting on age-plugin-"+name+"...", "browserpass")
	},
}

var promptTemplate = template.Must(template.New("script").Parse(`
var app = Application.currentApplication()
app.includeStandardAdditions = true
app.displayDialog(
	"{{ .Prompt }}", {
    defaultAnswer: "",
	withTitle: "age-plugin-{{ .Name }} prompt",
    buttons: ["Cancel", "OK"],
    defaultButton: "OK",
	cancelButton: "Cancel",
    hiddenAnswer: {{ .Hidden }},
})`))

func prompt(name, message string, secret bool) (string, error) {
	script := new(bytes.Buffer)
	if err := promptTemplate.Execute(script, map[string]interface{}{
		"Prompt": message, "Name": name, "Hidden": secret,
	}); err != nil {
		return "", err
	}

	c := exec.Command("osascript", "-s", "se", "-l", "JavaScript")
	c.Stdin = script
	out, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute osascript: %v", err)
	}
	var x struct {
		Result string `json:"textReturned"`
	}
	if err := json.Unmarshal(out, &x); err != nil {
		return "", fmt.Errorf("failed to parse osascript output: %v", err)
	}
	return x.Result, nil
}

var confirmTemplate = template.Must(template.New("script").Parse(`
var app = Application.currentApplication()
app.includeStandardAdditions = true
app.displayDialog(
	"{{ .Prompt }}", {
	withTitle: "age-plugin-{{ .Name }} prompt",
    buttons: ["Cancel", {{ if .No }} "{{ .No }}", {{ end }} "{{ .Yes }}"],
    defaultButton: "{{ .Yes }}",
	cancelButton: "Cancel",
})`))

func confirm(name, message, yes, no string) (choseYes bool, err error) {
	script := new(bytes.Buffer)
	if err := confirmTemplate.Execute(script, map[string]interface{}{
		"Prompt": message, "Name": name, "Yes": yes, "No": no,
	}); err != nil {
		return false, err
	}

	c := exec.Command("osascript", "-s", "se", "-l", "JavaScript")
	c.Stdin = script
	out, err := c.Output()
	if err != nil {
		return false, fmt.Errorf("failed to execute osascript: %v", err)
	}
	var x struct {
		Result string `json:"buttonReturned"`
	}
	if err := json.Unmarshal(out, &x); err != nil {
		return false, fmt.Errorf("failed to parse osascript output: %v", err)
	}
	return x.Result == yes, nil
}

func showNotification(message, title string) error {
	appleScript := `display notification %q with title %q`
	return exec.Command("osascript", "-e", fmt.Sprintf(appleScript, message, title)).Run()
}
