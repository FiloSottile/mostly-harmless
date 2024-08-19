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

func listDirectories(request *request) {
	responseData := response.MakeTreeResponse()

	for _, store := range request.Settings.Stores {
		defaultStorePath, err := getDefaultPasswordStorePath()
		if err != nil {
			log.Print("Unable to determine the location of the default password store: ", err)
			response.SendErrorAndExit(
				errors.CodeUnknownDefaultPasswordStoreLocation,
				&map[errors.Field]string{
					errors.FieldMessage: "Unable to determine the location of the default password store",
					errors.FieldAction:  "tree",
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
					errors.FieldAction:    "tree",
					errors.FieldStoreID:   store.ID,
					errors.FieldStoreName: store.Name,
					errors.FieldStorePath: store.Path,
				},
			)
		}

		var directories []string
		err = filepath.Walk(store.Path, func(path string, info fs.FileInfo, err error) error {
			if info.Mode().IsDir() && path != store.Path {
				if filepath.Base(path) == ".git" {
					return filepath.SkipDir
				}
				directories = append(directories, path)
			}
			return nil
		})
		if err != nil {
			log.Printf(
				"Unable to list the directory tree in the password store '%+v' at its location: %+v",
				store, err,
			)
			response.SendErrorAndExit(
				errors.CodeUnableToListDirectoriesInPasswordStore,
				&map[errors.Field]string{
					errors.FieldMessage:   "Unable to list the directory tree in the password store",
					errors.FieldAction:    "tree",
					errors.FieldError:     err.Error(),
					errors.FieldStoreID:   store.ID,
					errors.FieldStoreName: store.Name,
					errors.FieldStorePath: store.Path,
				},
			)
		}

		for i, directory := range directories {
			relativePath, err := filepath.Rel(store.Path, directory)
			if err != nil {
				log.Printf(
					"Unable to determine the relative path for a file '%v' in the password store '%+v': %+v",
					directory, store, err,
				)
				response.SendErrorAndExit(
					errors.CodeUnableToDetermineRelativeDirectoryPathInPasswordStore,
					&map[errors.Field]string{
						errors.FieldMessage:   "Unable to determine the relative path for a directory in the password store",
						errors.FieldAction:    "tree",
						errors.FieldError:     err.Error(),
						errors.FieldDirectory: directory,
						errors.FieldStoreID:   store.ID,
						errors.FieldStoreName: store.Name,
						errors.FieldStorePath: store.Path,
					},
				)
			}
			directories[i] = strings.Replace(relativePath, "\\", "/", -1) // normalize Windows paths
		}

		sort.Strings(directories)
		responseData.Directories[store.ID] = directories
	}

	response.SendOk(responseData)
}
