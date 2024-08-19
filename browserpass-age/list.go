package main

import (
	"io/fs"
	"log"
	"path/filepath"
	"sort"
	"strings"

	"github.com/browserpass/browserpass-native/errors"
	"github.com/browserpass/browserpass-native/response"
)

func listFiles(request *request) {
	responseData := response.MakeListResponse()

	for _, store := range request.Settings.Stores {
		defaultStorePath, err := getDefaultPasswordStorePath()
		if err != nil {
			log.Print("Unable to determine the location of the default password store: ", err)
			response.SendErrorAndExit(
				errors.CodeUnknownDefaultPasswordStoreLocation,
				&map[errors.Field]string{
					errors.FieldMessage: "Unable to determine the location of the default password store",
					errors.FieldAction:  "list",
					errors.FieldError:   err.Error(),
				},
			)
		}

		if store.Path != defaultStorePath {
			log.Printf(
				"The password store is not the default password store: %+v",
				store,
			)
			response.SendErrorAndExit(
				errors.CodeInaccessiblePasswordStore,
				&map[errors.Field]string{
					errors.FieldMessage:   "The password store is not accessible",
					errors.FieldAction:    "list",
					errors.FieldStoreID:   store.ID,
					errors.FieldStoreName: store.Name,
					errors.FieldStorePath: store.Path,
				},
			)
		}

		var files []string
		err = filepath.Walk(store.Path, func(path string, info fs.FileInfo, err error) error {
			if info.Mode().IsDir() {
				if filepath.Base(path) == ".git" {
					return filepath.SkipDir
				}
			} else {
				if strings.HasSuffix(path, ".age") {
					files = append(files, path)
				}
			}
			return nil
		})
		if err != nil {
			log.Printf(
				"Unable to list the files in the password store '%+v' at its location: %+v",
				store, err,
			)
			response.SendErrorAndExit(
				errors.CodeUnableToListFilesInPasswordStore,
				&map[errors.Field]string{
					errors.FieldMessage:   "Unable to list the files in the password store",
					errors.FieldAction:    "list",
					errors.FieldError:     err.Error(),
					errors.FieldStoreID:   store.ID,
					errors.FieldStoreName: store.Name,
					errors.FieldStorePath: store.Path,
				},
			)
		}

		for i, file := range files {
			relativePath, err := filepath.Rel(store.Path, file)
			if err != nil {
				log.Printf(
					"Unable to determine the relative path for a file '%v' in the password store '%+v': %+v",
					file, store, err,
				)
				response.SendErrorAndExit(
					errors.CodeUnableToDetermineRelativeFilePathInPasswordStore,
					&map[errors.Field]string{
						errors.FieldMessage:   "Unable to determine the relative path for a file in the password store",
						errors.FieldAction:    "list",
						errors.FieldError:     err.Error(),
						errors.FieldFile:      file,
						errors.FieldStoreID:   store.ID,
						errors.FieldStoreName: store.Name,
						errors.FieldStorePath: store.Path,
					},
				)
			}
			files[i] = strings.Replace(relativePath, "\\", "/", -1) // normalize Windows paths
		}

		sort.Strings(files)
		responseData.Files[store.ID] = files
	}

	response.SendOk(responseData)
}
