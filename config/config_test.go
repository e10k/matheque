package config

import (
	"bytes"
	"fmt"
	"testing"
)

func TestReadFile(t *testing.T) {
	var buffer bytes.Buffer
	text := `
KEY_01=VALUE_01
  WITH_SPACES = stuff  

# the above line is ignored and this one is ignored as well
in_quotes="quoted!" ignored
has_comments=actual_value5 # comment
HAS_HASH_IN_QUOTES="ab#cd" # comment
HAS_ESCAPED_QUOTES_AND_HASHES="ab#c\"de\#fg" #comment
HAS_ESCAPED_QUOTES_AND_HASHES_AND_SPACES = "  ab#c\"d e\#fg " # comment
HAS_VARIOUS_CHARS_IN_QUOTES = "±!@#$%^&*()\")~<>?|👁:" # comment
	`
	buffer.WriteString(text)

	values, err := readFile(&buffer)
	if err != nil {
		t.Error("Failed")
	}

	fmt.Println(values)

	if len(values) != 8 {
		t.Errorf("Expected %d values, got %d", 8, len(values))
	}

	tables := map[string]string{
		"KEY_01":                        `VALUE_01`,
		"WITH_SPACES":                   `stuff`,
		"in_quotes":                     `quoted!`,
		"has_comments":                  `actual_value5`,
		"HAS_HASH_IN_QUOTES":            `ab#cd`,
		"HAS_ESCAPED_QUOTES_AND_HASHES": `ab#c\"de\#fg`,
		"HAS_ESCAPED_QUOTES_AND_HASHES_AND_SPACES": `ab#c\"d e\#fg`,
		"HAS_VARIOUS_CHARS_IN_QUOTES":              `±!@#$%^&*()\")~<>?|👁:`,
	}

	for k, v := range tables {
		got, ok := values[k]

		if !ok {
			t.Errorf("Key not found: %q", k)
		}

		if got != v {
			t.Errorf("Expected %q, got %q", v, got)
		}
	}
}
