//go:build linux

package scenes

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func pickJSONFileByOS(startDir string) (string, error) {
	cmd := exec.Command("zenity", "--file-selection", "--filename="+startDir, "--title=Importa problema piano inclinato", "--file-filter=JSON files | *.json")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", errImportCancelled
		}
		if errors.Is(err, exec.ErrNotFound) {
			return "", errors.New("file picker non disponibile: installa zenity")
		}
		return "", fmt.Errorf("impossibile aprire il gestore file: %w", err)
	}

	path := strings.TrimSpace(string(out))
	if path == "" {
		return "", errImportCancelled
	}

	return path, nil
}
