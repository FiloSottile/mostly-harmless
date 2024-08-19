package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
	"filippo.io/age/plugin"
	"github.com/browserpass/browserpass-native/errors"
	"github.com/browserpass/browserpass-native/response"
)

func fetchDecryptedContents(request *request) {
	responseData := response.MakeFetchResponse()

	if !strings.HasSuffix(request.File, ".age") {
		log.Printf("The requested password file '%v' does not have the expected '.age' extension", request.File)
		response.SendErrorAndExit(
			errors.CodeInvalidPasswordFileExtension,
			&map[errors.Field]string{
				errors.FieldMessage: "The requested password file does not have the expected '.age' extension",
				errors.FieldAction:  "fetch",
				errors.FieldFile:    request.File,
			},
		)
	}

	store, ok := request.Settings.Stores[request.StoreID]
	if !ok {
		log.Printf(
			"The password store with ID '%v' is not present in the list of stores '%+v'",
			request.StoreID, request.Settings.Stores,
		)
		response.SendErrorAndExit(
			errors.CodeInvalidPasswordStore,
			&map[errors.Field]string{
				errors.FieldMessage: "The password store is not present in the list of stores",
				errors.FieldAction:  "fetch",
				errors.FieldStoreID: request.StoreID,
			},
		)
	}

	defaultStorePath, err := getDefaultPasswordStorePath()
	if err != nil {
		log.Print("Unable to determine the location of the default password store: ", err)
		response.SendErrorAndExit(
			errors.CodeUnknownDefaultPasswordStoreLocation,
			&map[errors.Field]string{
				errors.FieldMessage: "Unable to determine the location of the default password store",
				errors.FieldAction:  "fetch",
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
				errors.FieldAction:    "fetch",
				errors.FieldStoreID:   store.ID,
				errors.FieldStoreName: store.Name,
				errors.FieldStorePath: store.Path,
			},
		)
	}

	if !filepath.IsLocal(request.File) {
		log.Printf("The requested password file '%v' is not a local file", request.File)
		response.SendErrorAndExit(
			errors.CodeInvalidPasswordFileExtension,
			&map[errors.Field]string{
				errors.FieldMessage:   "The requested password file is not a local file",
				errors.FieldAction:    "fetch",
				errors.FieldFile:      request.File,
				errors.FieldStoreID:   store.ID,
				errors.FieldStoreName: store.Name,
				errors.FieldStorePath: store.Path,
			},
		)
	}

	if err := showNotification("Fetching "+request.File, "browserpass"); err != nil {
		log.Printf("Unable to show the notification: %+v", err)
		response.SendErrorAndExit(
			errors.CodeUnableToDecryptPasswordFile,
			&map[errors.Field]string{
				errors.FieldMessage:   "Unable to show the notification",
				errors.FieldAction:    "fetch",
				errors.FieldError:     err.Error(),
				errors.FieldFile:      request.File,
				errors.FieldStoreID:   store.ID,
				errors.FieldStoreName: store.Name,
				errors.FieldStorePath: store.Path,
			},
		)
	}
	responseData.Contents, err = decryptFile(filepath.Join(store.Path, request.File))
	if err != nil {
		log.Printf(
			"Unable to decrypt the password file '%v' in the password store '%+v': %+v",
			request.File, store, err,
		)
		response.SendErrorAndExit(
			errors.CodeUnableToDecryptPasswordFile,
			&map[errors.Field]string{
				errors.FieldMessage:   "Unable to decrypt the password file",
				errors.FieldAction:    "fetch",
				errors.FieldError:     err.Error(),
				errors.FieldFile:      request.File,
				errors.FieldStoreID:   store.ID,
				errors.FieldStoreName: store.Name,
				errors.FieldStorePath: store.Path,
			},
		)
	}

	response.SendOk(responseData)
}

func decryptFile(file string) (string, error) {
	identitiesFile, err := getDefaultIdentitiesFile()
	if err != nil {
		return "", err
	}

	ids, err := os.Open(identitiesFile)
	if err != nil {
		return "", err
	}
	defer ids.Close()

	identities, err := parseIdentities(ids)
	if err != nil {
		return "", fmt.Errorf("failed to parse identities: %v", err)
	}

	ff, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer ff.Close()

	f, err := age.Decrypt(ff, identities...)
	if err != nil {
		return "", err
	}

	contents, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(contents), nil
}

func getDefaultIdentitiesFile() (string, error) {
	path := os.Getenv("PASSAGE_IDENTITIES_FILE")
	if path != "" {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".passage", "identities"), nil
}

// parseIdentities and parseIdentity are copied from cmd/age.
func parseIdentities(f io.Reader) ([]age.Identity, error) {
	const privateKeySizeLimit = 1 << 24 // 16 MiB
	var ids []age.Identity
	scanner := bufio.NewScanner(io.LimitReader(f, privateKeySizeLimit))
	var n int
	for scanner.Scan() {
		n++
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		i, err := parseIdentity(line)
		if err != nil {
			return nil, fmt.Errorf("error at line %d: %v", n, err)
		}
		ids = append(ids, i)

	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read secret keys file: %v", err)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no secret keys found")
	}
	return ids, nil
}

func parseIdentity(s string) (age.Identity, error) {
	switch {
	case strings.HasPrefix(s, "AGE-PLUGIN-"):
		return plugin.NewIdentity(s, pluginHeadlessUI)
	case strings.HasPrefix(s, "AGE-SECRET-KEY-1"):
		return age.ParseX25519Identity(s)
	default:
		return nil, fmt.Errorf("unknown identity type")
	}
}
