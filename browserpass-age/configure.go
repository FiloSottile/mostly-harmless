package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/browserpass/browserpass-native/errors"
	"github.com/browserpass/browserpass-native/response"
)

func configure(request *request) {
	responseData := response.MakeConfigureResponse()

	if request.Settings.GpgPath != "" {
		log.Print("Custom gpg paths are not supported")
		response.SendErrorAndExit(
			errors.CodeInvalidGpgPath,
			&map[errors.Field]string{
				errors.FieldMessage: "Custom gpg paths are not supported",
				errors.FieldAction:  "configure",
			},
		)
	}

	if len(request.Settings.Stores) != 0 {
		log.Print("Custom stores are not supported")
		response.SendErrorAndExit(
			errors.CodeInvalidPasswordStore,
			&map[errors.Field]string{
				errors.FieldMessage: "Custom stores are not supported",
				errors.FieldAction:  "configure",
			},
		)
	}

	var err error
	responseData.DefaultStore.Path, err = getDefaultPasswordStorePath()
	if err != nil {
		log.Print("Unable to determine the location of the default password store: ", err)
		response.SendErrorAndExit(
			errors.CodeUnknownDefaultPasswordStoreLocation,
			&map[errors.Field]string{
				errors.FieldMessage: "Unable to determine the location of the default password store",
				errors.FieldAction:  "configure",
				errors.FieldError:   err.Error(),
			},
		)
	}

	responseData.DefaultStore.Settings, err = readDefaultSettings(responseData.DefaultStore.Path)
	if err != nil {
		log.Printf(
			"Unable to read .browserpass.json of the default password store in '%v': %+v",
			responseData.DefaultStore.Path, err,
		)
		response.SendErrorAndExit(
			errors.CodeUnreadableDefaultPasswordStoreDefaultSettings,
			&map[errors.Field]string{
				errors.FieldMessage:   "Unable to read .browserpass.json of the default password store",
				errors.FieldAction:    "configure",
				errors.FieldError:     err.Error(),
				errors.FieldStorePath: responseData.DefaultStore.Path,
			},
		)
	}

	response.SendOk(responseData)
}

func getDefaultPasswordStorePath() (string, error) {
	path := os.Getenv("PASSAGE_DIR")
	if path != "" {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".passage", "store"), nil
}

func readDefaultSettings(storePath string) (string, error) {
	content, err := os.ReadFile(filepath.Join(storePath, ".browserpass.json"))
	if os.IsNotExist(err) {
		return "{}", nil
	}
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(content, &StoreSettings{}); err != nil {
		return "", err
	}
	return string(content), nil
}
