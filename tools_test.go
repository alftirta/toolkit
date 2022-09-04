package toolkit

import (
	"bytes"
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
	{
		name:          "allowed rename",
		allowedTypes:  []string{"image/jpeg", "image/png"},
		renameFile:    true,
		errorExpected: false,
	},
	{
		name:          "not allowed",
		allowedTypes:  []string{"image/jpeg"},
		renameFile:    false,
		errorExpected: true,
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

func TestTools_UploadOneFile(t *testing.T) {
	// set up a pipe to avoid buffering
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
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

	uploadedFile, err := testTools.UploadOneFile(request, "./testdata/uploads")
	if err != nil {
		t.Error("error uploading files", err)
	}

	if _, err = os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFileName)); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", err.Error())
	}

	// clean up
	if err = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFileName)); err != nil {
		t.Error("error removing file", err)
	}
}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	var testTool Tools

	// test when the directory does not exist
	if err := testTool.CreateDirIfNotExist("./testdata/testdir"); err != nil {
		t.Error(err)
	}

	// test when the directory does exist
	if err := testTool.CreateDirIfNotExist("./testdata/testdir"); err != nil {
		t.Error(err)
	}

	if err := os.Remove("./testdata/testdir"); err != nil {
		t.Error(err)
	}
}

var slugTests = []struct {
	name          string
	s             string
	expected      string
	errorExpected bool
}{
	{
		name:          "valid string",
		s:             "now is the time",
		expected:      "now-is-the-time",
		errorExpected: false,
	},
	{
		name:          "empty string",
		s:             "",
		expected:      "",
		errorExpected: true,
	},
	{
		name:          "complex string",
		s:             "Now is the time for all GOOD men! + fish & such &^123",
		expected:      "now-is-the-time-for-all-good-men-fish-such-123",
		errorExpected: false,
	},
	{
		name:          "japanese string",
		s:             "こんにちは世界",
		expected:      "",
		errorExpected: true,
	},
	{
		name:          "japanese string and roman characters",
		s:             "hello world こんにちは世界",
		expected:      "hello-world",
		errorExpected: false,
	},
}

func TestTools_Slugify(t *testing.T) {
	var testTool Tools
	for _, test := range slugTests {
		slugified, err := testTool.Slugify(test.s)
		if err != nil && !test.errorExpected {
			t.Errorf("%s: error received when none expected: %s", test.name, err.Error())
		}
		if !test.errorExpected && slugified != test.expected {
			t.Errorf("%s: wrong slug returned; expected %s but got %s", test.name, test.expected, slugified)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/arbitrary-endpoint", nil)

	var testTool Tools
	testTool.DownloadStaticFile(rec, req, "./testdata", "pic.jpg", "puppy.jpg")

	res := rec.Result()
	defer res.Body.Close()

	if res.Header["Content-Length"][0] != "98827" {
		t.Error("wrong content length of", res.Header["Content-Length"][0])
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"puppy.jpg\"" {
		t.Error("wrong content disposition")
	}

	if _, err := io.ReadAll(res.Body); err != nil {
		t.Error(err)
	}
}

var readJSONTests = []struct {
	name               string
	json               string
	errorExpected      bool
	maxJSONSize        int64
	allowUnknownFields bool
}{
	{
		name:               "good json",
		json:               `{"foo": "bar"}`,
		errorExpected:      false,
		maxJSONSize:        1024,
		allowUnknownFields: false,
	},
}

func TestTools_ReadJSON(t *testing.T) {
	var testTool Tools
	for _, test := range readJSONTests {
		// set the max file size
		testTool.MaxJSONSize = test.maxJSONSize

		// allow/disallow unknown fields
		testTool.AllowUnknownFields = test.allowUnknownFields

		// declare a variable to read the decoded json into
		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		// create a request with the body
		req := httptest.NewRequest("POST", "/arbitrary-endpoint", bytes.NewReader([]byte(test.json)))
		defer req.Body.Close()

		// create a recorder
		rr := httptest.NewRecorder()

		// test the ReadJSON method
		err := testTool.ReadJSON(rr, req, &decodedJSON)
		if test.errorExpected && err == nil {
			t.Errorf("%s: error expected, but none received", test.name)
		}
		if !test.errorExpected && err != nil {
			t.Errorf("%s: error not expected, but one received: %s", test.name, err.Error())
		}
	}
}
