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
