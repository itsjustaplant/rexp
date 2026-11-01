package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// EditorPreference reads the optional tool settings without creating them.
func EditorPreference() (string, error) {
	projects, err := ProjectsDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(filepath.Dir(projects), "config.yaml")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read tool configuration %q: %w", path, err)
	}
	var document yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&document); errors.Is(err, io.EOF) {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("decode tool configuration %q: %w", path, err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("tool configuration %q must contain exactly one YAML document", path)
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return "", fmt.Errorf("tool configuration %q must be a mapping", path)
	}
	mapping := document.Content[0]
	for i := 0; i < len(mapping.Content); i += 2 {
		key, value := mapping.Content[i], mapping.Content[i+1]
		if key.Tag != "!!str" || key.Value != "editor" || value.Kind != yaml.ScalarNode || value.Tag != "!!str" {
			return "", fmt.Errorf("tool configuration %q only supports editor as a string", path)
		}
	}
	var settings struct {
		Editor string `yaml:"editor"`
	}
	strict := yaml.NewDecoder(bytes.NewReader(data))
	strict.KnownFields(true)
	if err := strict.Decode(&settings); err != nil {
		return "", fmt.Errorf("decode tool configuration %q: %w", path, err)
	}
	return settings.Editor, nil
}
