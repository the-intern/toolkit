package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

/* See https://pkg.go.dev/testing */

func Test_Tools_RandomString(t *testing.T) {

	var testTools Tools

	ln := 10
	s := testTools.RandomString(ln)

	if len(s) != ln {
		t.Error("incorrect length of random string returned")
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

func Test_Tools_UploadFiles(t *testing.T) {

	for _, entry := range uploadTests {
		// set up a pipe to avoid buffering
		// Pipe creates a synchronous in-memory pipe. It can be used to connect code expecting an io.Reader with code expecting an io.Writer.
		pipe_reader, pipe_write := io.Pipe()
		writer := multipart.NewWriter(pipe_write)

		// have to create a wait group, a struct from the sync library
		waitGroup := sync.WaitGroup{}
		waitGroup.Add(1)

		// inline func to run concurrently
		go func() {
			defer writer.Close()
			defer waitGroup.Done()

			// create the form data field 'file'
			// part ==> io.Writer object
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error(err)
			}

			f, err := os.Open("./testdata/img.png")

			if err != nil {
				t.Error(err)
			}
			defer f.Close()

			// now decode the image
			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image", err)
			}

			// remember part == io.Writer; img == image.Image
			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()
		// concurrently (?) read from the pipe which receives data
		request := httptest.NewRequest("POST", "/", pipe_reader)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = entry.allowedTypes

		uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", entry.renameFile)
		if err != nil && !entry.errorExpected {
			t.Error(err)
		}

		if !entry.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {

				t.Errorf("%s: expected file to exist: %s", entry.name, err.Error())

			}

			// cleanup resources post test
			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))
		}

		if !entry.errorExpected && err != nil {
			t.Errorf("%s: error expected but none received", entry.name)
		}

		waitGroup.Wait()
	}
}

func Test_Tools_UploadOneFile(t *testing.T) {
	// set up a pipe to avoid buffering
	// Pipe creates a synchronous in-memory pipe. It can be used to connect code expecting an io.Reader with code expecting an io.Writer.
	pipe_reader, pipe_write := io.Pipe()
	writer := multipart.NewWriter(pipe_write)

	// inline func to run concurrently
	go func() {
		defer writer.Close()

		// create the form data field 'file'
		// part ==> io.Writer object
		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err != nil {
			t.Error(err)
		}

		f, err := os.Open("./testdata/img.png")

		if err != nil {
			t.Error(err)
		}
		defer f.Close()

		// now decode the image
		img, _, err := image.Decode(f)
		if err != nil {
			t.Error("error decoding image", err)
		}

		// remember part == io.Writer; img == image.Image
		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}
	}()
	// concurrently (?) read from the pipe which receives data
	request := httptest.NewRequest("POST", "/", pipe_reader)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	var testTools Tools

	uploadedFiles, err := testTools.UploadOneFile(request, "./testdata/uploads/", true)
	if err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName)); os.IsNotExist(err) {

		t.Errorf("expected file to exist: %s", err.Error())

	}

	// cleanup resources post test
	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName))

}

func Test_Tools_CreateDirIfNotExist(t *testing.T) {
	var testTool Tools

	err := testTool.CreateDirIfNotExist("./testdata/myDir")

	if err != nil {
		t.Error(err)
	}

	err = testTool.CreateDirIfNotExist("./testdata/myDir")

	if err != nil {
		t.Error(err)
	}

	_ = os.Remove("./testdata/myDir")
}

var slugTests = []struct {
	name           string
	s              string
	expectedResult string
	errorExpected  bool
}{
	{
		name:           "valid string",
		s:              "now is the time",
		expectedResult: "now-is-the-time",
		errorExpected:  false,
	},
	{
		name:           "empty string",
		s:              "",
		expectedResult: "",
		errorExpected:  true,
	},
	{
		name:           "complex string",
		s:              "Now is the Time For ALL GOOD men! + fish & &^123",
		expectedResult: "now-is-the-time-for-all-good-men-fish-123",
		errorExpected:  false,
	},
	{
		name:           "japanese string",
		s:              "こんにちは世界",
		expectedResult: "",
		errorExpected:  true,
	},
	{
		name:           "japanese string and roman characters",
		s:              "こんにちは世界 now is the time",
		expectedResult: "now-is-the-time",
		errorExpected:  false,
	},
}

func Test_Tools_Slugify(t *testing.T) {
	var testTool Tools

	for _, entry := range slugTests {
		slug, err := testTool.Slugify(entry.s)

		if err != nil && !entry.errorExpected {
			t.Errorf("%s: error received when none expected: %s", entry.name, err.Error())
		}

		if !entry.errorExpected && slug != entry.expectedResult {
			t.Errorf("%s: wrong slug returned; expected %s but got %s", entry.name, entry.expectedResult, slug)
		}
	}
}

func Test_Tools_DownloadStaticFile(t *testing.T) {
	// request and response
	responseReporter := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	var testTool Tools

	testTool.DownloadStaticFile(responseReporter, req, "./testdata", "pic.png", "testImage.png")

	result := responseReporter.Result()
	defer result.Body.Close()

	if result.Header["Content-Length"][0] != "73078" {
		t.Error("wrong file content length of", result.Header["Content-Length"][0])
	}

	if result.Header["Content-Disposition"][0] != "attachment; filename=\"testImage.png\"" {
		t.Error("wrong content disposition")
	}

	_, err := io.ReadAll(result.Body)

	if err != nil {
		t.Error(err)
	}
}

// json testing

var jsonTests = []struct {
	name          string
	json          string
	errorExpected bool
	maxSize       int
	allowUnknown  bool
}{
	{name: "good JSON", json: `{"foo":"bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: false},
	{name: "badly formatted JSON", json: `{"foo":}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "incorrect type", json: `{"foo": 1}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "two json files", json: `{"foo": "1"}{"alpha":"beta"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "empty body", json: ``, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "syntax error in json", json: `{"foo": 1"`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "unknown field in json", json: `{"fooo": "1"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "allow unknown field in json", json: `{"fooo": "1"}`, errorExpected: false, maxSize: 1024, allowUnknown: true},
	{name: "missing field name", json: `{jack: "1"}`, errorExpected: true, maxSize: 1024, allowUnknown: true},
	{name: "file too large", json: `{"foo": "bar"}`, errorExpected: true, maxSize: 5, allowUnknown: true},
	{name: "not json", json: `Hello, World`, errorExpected: true, maxSize: 1024, allowUnknown: true},
}

func Test_Tools_ReadJSON(t *testing.T) {
	var testTool Tools

	for _, entry := range jsonTests {
		// set max file size
		testTool.MaxJSONSize = entry.maxSize
		// allowable or not fields
		testTool.AllowUnknownFields = entry.allowUnknown
		// declare variable to read the decoded json into
		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		// create a request with a json body
		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(entry.json)))

		if err != nil {
			t.Log("Error", err)
		}

		// now with a request, need a response recorder - taking place of a response writer
		rr := httptest.NewRecorder()

		err = testTool.ReadJSON(rr, req, &decodedJSON)

		if entry.errorExpected && err == nil {
			t.Errorf("%s: error expected but none received", entry.name)
		}

		if !entry.errorExpected && err != nil {
			t.Errorf("%s: error not expected but one received: %s", entry.name, err.Error())
		}

		req.Body.Close()
	}
}

func Test_Tools_WriteJSON(t *testing.T) {
	var testTool Tools

	rr := httptest.NewRecorder()
	payload := JSONResponse{
		Error:   false,
		Message: "foo",
	}

	headers := make(http.Header)
	headers.Add("FOO", "BAR")

	err := testTool.WriteJSON(rr, http.StatusOK, payload, headers)

	if err != nil {
		t.Errorf("failed to write JSON: %v", err)
	}

}

func Test_Tools_ErrorJSON(t *testing.T) {

	var testTool Tools

	rr := httptest.NewRecorder()
	err := testTool.ErrorJSON(rr, errors.New("some error or other"), http.StatusServiceUnavailable)

	if err != nil {
		t.Error(err)
	}

	var payload JSONResponse
	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&payload)

	if err != nil {
		t.Error("received error when decoding json", err)
	}

	if !payload.Error {
		t.Error("error set to false in json but should be true")
	}

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("wrong status code returned; expected 503 but got %d", rr.Code)
	}

}
