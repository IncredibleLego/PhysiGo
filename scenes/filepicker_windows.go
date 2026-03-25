//go:build windows

package scenes

import (
	"errors"
	"strings"

	"github.com/sqweek/dialog"
)

func pickJSONFileByOS(startDir string) (string, error) {
	path, err := dialog.File().Filter("JSON files", "json").SetStartDir(startDir).Title("Importa problema piano inclinato").Load()
	if err != nil {
		if errors.Is(err, dialog.Cancelled) {
			return "", errImportCancelled
		}
		return "", err
	}

	path = strings.TrimSpace(path)
	if path == "" {
		return "", errImportCancelled
	}

	return path, nil
}
