package toolkit

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const randomStringSource string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

// Tools is the type used to instantiate this module.
// Any variable of this type will have access too all the methods with the receiver *Tools.
type Tools struct {
	MaxFileSize      int64
	AllowedFileTypes []string
}

// RandomString returns a string of random characters of length n,
// using randomStringSource as the source for the string.
func (t *Tools) RandomString(n int) string {
	s, r := make([]rune, n), []rune(randomStringSource)
	for i := range s {
		p, err := rand.Prime(rand.Reader, len(r))
		if err != nil {
			return "RandomString Error"
		}
		x, y := p.Uint64(), uint64(len(r))
		s[i] = r[x%y]
	}
	return string(s)
}

// UploadedFile is a struct used to save information about an uploaded file.
type UploadedFile struct {
	NewFileName      string
	OriginalFileName string
	FileSize         int64
}

func (t *Tools) UploadFiles(r *http.Request, uploadDir string, rename ...bool) ([]*UploadedFile, error) {
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}

	var uploadedFiles []*UploadedFile
	var err error
	if t.MaxFileSize == 0 {
		t.MaxFileSize = 1024 * 1024 * 1024 // 1kB * 1kB * 1kB == 1GB
	}
	if err = r.ParseMultipartForm(t.MaxFileSize); err != nil {
		return nil, errors.New("the uploaded file is too big")
	}

	for _, fileHeaders := range r.MultipartForm.File {
		for _, fileHeader := range fileHeaders {
			uploadedFiles, err = func(uploadedFiles []*UploadedFile) ([]*UploadedFile, error) {
				var uploadedFile UploadedFile
				inFile, err := fileHeader.Open()
				if err != nil {
					return nil, err
				}
				defer inFile.Close()

				// look at the first 512 bytes of the file in order to figure out what it is
				buff := make([]byte, 512)

				// get the first 512 bytes of the file
				if _, err = inFile.Read(buff); err != nil {
					return nil, err
				}

				// check to see if the file type is permitted
				allowed := false
				fileType := http.DetectContentType(buff) // "image/jpeg" || "image/png" || "image/gif" || etc.

				if len(t.AllowedFileTypes) > 0 {
					for _, allowedFileType := range t.AllowedFileTypes {
						if strings.EqualFold(fileType, allowedFileType) {
							allowed = true
							break
						}
					}
				} else {
					allowed = true
				}

				if !allowed {
					return nil, errors.New("the uploaded file type is not permitted")
				}

				if _, err = inFile.Seek(0, 0); err != nil {
					return nil, err
				}

				if renameFile {
					uploadedFile.NewFileName = fmt.Sprintf("%s%s", t.RandomString(25), filepath.Ext(fileHeader.Filename))
				} else {
					uploadedFile.NewFileName = fileHeader.Filename
				}

				var outFile *os.File
				defer outFile.Close()

				if outFile, err = os.Create(filepath.Join(uploadDir, uploadedFile.NewFileName)); err != nil {
					return nil, err
				}

				if fileSize, err := io.Copy(outFile, inFile); err != nil {
					return nil, err
				} else {
					uploadedFile.FileSize = fileSize
				}

				uploadedFiles = append(uploadedFiles, &uploadedFile)
				return uploadedFiles, err
			}(uploadedFiles)
			if err != nil {
				return uploadedFiles, err
			}
		}
	}

	return uploadedFiles, err
}
