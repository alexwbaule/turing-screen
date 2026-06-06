package theme

import (
	"bytes"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// LoadTheme lê um arquivo theme.yaml e o decodifica para a struct Theme.
func LoadTheme(filePath string) (*Theme, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var theme Theme
	err = yaml.Unmarshal(data, &theme)
	if err != nil {
		return nil, err
	}

	return &theme, nil
}

// SaveTheme codifica uma struct Theme para YAML e a salva em um arquivo.
// Uses a custom post-processor to remove unnecessary quotes from keys like "Y".
func SaveTheme(theme *Theme, filePath string) error {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(theme); err != nil {
		return err
	}
	encoder.Close()

	// Post-process: remove quotes from YAML keys that got quoted unnecessarily
	// e.g. "Y": 20 -> Y: 20, "N": false -> N: false
	result := unquoteYAMLKeys(buf.Bytes())

	return os.WriteFile(filePath, result, 0644)
}

// unquoteYAMLKeys removes unnecessary quotes from YAML keys.
// The yaml.v3 encoder quotes keys like Y, N, Yes, No because they're
// booleans in YAML 1.1. We force them unquoted since our consumer (Viper) handles it.
var quotedKeyRegex = regexp.MustCompile(`(?m)^(\s*)"([A-Za-z_][A-Za-z_0-9]*)":`)

func unquoteYAMLKeys(data []byte) []byte {
	return quotedKeyRegex.ReplaceAll(data, []byte("${1}${2}:"))
}
