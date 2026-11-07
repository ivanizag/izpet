package izpet

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"embed"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	internalPrefix = "<internal>/"
	embedPrefix    = "resources/"
	httpPrefix     = "http://"
	httpsPrefix    = "https://"
)

//go:embed resources
var internalFiles embed.FS

func isInternalResource(filename string) bool {
	return strings.HasPrefix(filename, internalPrefix)
}

func isHTTPResource(filename string) bool {
	return strings.HasPrefix(filename, httpPrefix) ||
		strings.HasPrefix(filename, httpsPrefix)
}

func normalizeFilename(filename string) string {
	// Remove quotes if surrounded by them
	if strings.HasPrefix(filename, "\"") && strings.HasSuffix(filename, "\"") {
		filename = filename[1 : len(filename)-1]
	}

	// Expand the tilde if prefixed by it
	if strings.HasPrefix(filename, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			filename = home + filename[1:]
		}
	}
	return filename
}

// LoadResource loads in memory a file from the filesystem, http or the
// embedded resources. Gzip and zip files are uncompressed, the first file of
// a zip is taken.
func LoadResource(filename string) ([]uint8, error) {
	filename = normalizeFilename(filename)

	var file io.Reader
	if isInternalResource(filename) {
		resource := embedPrefix + strings.TrimPrefix(filename, internalPrefix)
		resourceFile, err := internalFiles.Open(resource)
		if err != nil {
			return nil, err
		}
		defer resourceFile.Close()
		file = resourceFile

	} else if isHTTPResource(filename) {
		response, err := http.Get(filename)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		file = response.Body

	} else {
		diskFile, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		defer diskFile.Close()
		file = diskFile
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	contentType := http.DetectContentType(data)
	if contentType == "application/x-gzip" {
		gz, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		return io.ReadAll(gz)

	} else if contentType == "application/zip" {
		z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, err
		}
		for _, zf := range z.File {
			if zf.FileInfo().IsDir() {
				continue
			}
			f, err := zf.Open()
			if err != nil {
				return nil, err
			}
			defer f.Close()
			return io.ReadAll(f)
		}
		return nil, errors.New("empty zip file")
	}

	return data, nil
}
