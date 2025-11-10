// Package signal contains helpers to exchange the SDP session
// description between examples.
package signal

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"time"
)

// MustReadFromFile waits until the file exists and has content, then returns it
func MustReadFromFile(filename string) string {
	for {
		data, err := os.ReadFile(filename)
		if err == nil && len(data) > 0 {
			// File exists and has content
			content := strings.TrimSpace(string(data))
			if len(content) > 0 {
				return content
			}
		}
		// Wait a bit before checking again
		time.Sleep(100 * time.Millisecond)
	}
}

// ClearFile clears the contents of a file
func ClearFile(filename string) {
	os.WriteFile(filename, []byte{}, 0644)
}

// Encode encodes the input in base64
func Encode(obj interface{}) string {
	b, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// Decode decodes the input from base64
func Decode(in string, obj interface{}) {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(b, obj)
	if err != nil {
		panic(err)
	}
}
