package main

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/browserpass/browserpass-native/errors"
	"github.com/browserpass/browserpass-native/response"
)

type StoreSettings struct {
	GpgPath string `json:"gpgPath"`
}

type store struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Path     string        `json:"path"`
	Settings StoreSettings `json:"settings"`
}

type settings struct {
	GpgPath string           `json:"gpgPath"`
	Stores  map[string]store `json:"stores"`
}

type request struct {
	Action       string      `json:"action"`
	Settings     settings    `json:"settings"`
	File         string      `json:"file"`
	Contents     string      `json:"contents"`
	StoreID      string      `json:"storeId"`
	EchoResponse interface{} `json:"echoResponse"`
}

func main() {
	env, err := exec.Command("zsh", "--interactive", "-c", "env").Output()
	if err != nil {
		log.Print("Unable to get the environment variables: ", err)
		response.SendErrorAndExit(
			errors.CodeUnableToDetectGpgPath,
			&map[errors.Field]string{
				errors.FieldMessage: "Unable to get the environment variables",
				errors.FieldError:   err.Error(),
			},
		)
	}
	for _, line := range strings.Split(string(env), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch k {
		case "PATH":
			// For age's plugin detection.
			os.Setenv(k, v)
		case "PASSAGE_IDENTITIES_FILE":
			os.Setenv(k, v)
		case "PASSAGE_DIR":
			os.Setenv(k, v)
		}
	}

	requestLength, err := parseRequestLength(os.Stdin)
	if err != nil {
		log.Print("Unable to parse the length of the browser request: ", err)
		response.SendErrorAndExit(
			errors.CodeParseRequestLength,
			&map[errors.Field]string{
				errors.FieldMessage: "Unable to parse the length of the browser request",
				errors.FieldError:   err.Error(),
			},
		)
	}

	request, err := parseRequest(requestLength, os.Stdin)
	if err != nil {
		log.Print("Unable to parse the browser request: ", err)
		response.SendErrorAndExit(
			errors.CodeParseRequest,
			&map[errors.Field]string{
				errors.FieldMessage: "Unable to parse the browser request",
				errors.FieldError:   err.Error(),
			},
		)
	}

	switch request.Action {
	case "configure":
		configure(request)
	case "list":
		listFiles(request)
	case "tree":
		listDirectories(request)
	case "fetch":
		fetchDecryptedContents(request)
	case "echo":
		response.SendRaw(request.EchoResponse)
	default:
		log.Printf("Received a browser request with an unknown action: %+v", request)
		response.SendErrorAndExit(
			errors.CodeInvalidRequestAction,
			&map[errors.Field]string{
				errors.FieldMessage: "Invalid request action",
				errors.FieldAction:  request.Action,
			},
		)
	}
}

func parseRequestLength(input io.Reader) (uint32, error) {
	var length uint32
	if err := binary.Read(input, binary.LittleEndian, &length); err != nil {
		return 0, err
	}
	return length, nil
}

func parseRequest(messageLength uint32, input io.Reader) (*request, error) {
	var parsed request
	reader := &io.LimitedReader{R: input, N: int64(messageLength)}
	if err := json.NewDecoder(reader).Decode(&parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}
