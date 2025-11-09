// Package signal contains helpers to exchange the SDP session
// description between examples.
package signal

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

// Allows compressing offer/answer to bypass terminal input limits.
const compress = false

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
// It can optionally zip the input before encoding
func Encode(obj interface{}) string {
	b, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	if compress {
		b = zip(b)
	}

	return base64.StdEncoding.EncodeToString(b)
}

// Decode decodes the input from base64
// It can optionally unzip the input after decoding
func Decode(in string, obj interface{}) {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		panic(err)
	}

	if compress {
		b = unzip(b)
	}

	err = json.Unmarshal(b, obj)
	if err != nil {
		panic(err)
	}
}

func zip(in []byte) []byte {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	_, err := gz.Write(in)
	if err != nil {
		panic(err)
	}
	err = gz.Flush()
	if err != nil {
		panic(err)
	}
	err = gz.Close()
	if err != nil {
		panic(err)
	}
	return b.Bytes()
}

func unzip(in []byte) []byte {
	var b bytes.Buffer
	_, err := b.Write(in)
	if err != nil {
		panic(err)
	}
	r, err := gzip.NewReader(&b)
	if err != nil {
		panic(err)
	}
	res, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return res
}
