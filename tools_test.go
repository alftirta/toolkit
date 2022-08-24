package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testTools Tools
	s := testTools.RandomString(10)
	if len(s) != 10 {
		t.Error("wrong length of random string")
	}
}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{
		name:          "allowed no rename",
		allowedTypes:  []string{"image/jpeg", "image/png"},
		renameFile:    false,
		errorExpected: false,
	},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, uploadTest := range uploadTests {
		// set up a pipe to avoid buffering
		pr, pw := io.Pipe() // *PipeReader, *PipeWriter
		writer := multipart.NewWriter(pw)

		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			defer writer.Close()

			// create the form data field, let's say "pngFile"
			pngFileWriter, err := writer.CreateFormFile("pngFile", "./testdata/img.png")
			if err != nil {
				t.Error("error creating form file", err)
			}

			pngFileReader, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error("error opening img.png file", err)
			}
			defer pngFileReader.Close()

			img, _, err := image.Decode(pngFileReader)
			if err != nil {
				t.Error("error decoding image", err)
			}

			if err = png.Encode(pngFileWriter, img); err != nil {
				t.Error("error encoding png")
			}
		}()

		// read from the pipe which receives data
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = uploadTest.allowedTypes

		uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads", uploadTest.renameFile)
		if err != nil && !uploadTest.errorExpected {
			t.Error("error uploading files", err)
		}

		if !uploadTest.errorExpected {
			if _, err = os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s: expected file to exist: %s", uploadTest.name, err.Error())
			}

			// clean up
			if err = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); err != nil {
				t.Error("error removing file", err)
			}
		}

		if !uploadTest.errorExpected && err != nil {
			t.Errorf("%s: error expected but none received", uploadTest.name)
		}

		wg.Wait()
	}
}
