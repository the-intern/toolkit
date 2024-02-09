package toolkit

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Tools is the type used to instantiate this module.  Any variable of this type has access to the methods defined with the receiver *Tools
type Tools struct {
	MaxFileSize        int
	AllowedFileTypes   []string
	MaxJSONSize        int
	AllowUnknownFields bool
}

// RandomString returns a random string of characters of length n, using as its source randomStringSource for the possible characters to comprise the return value
func (t *Tools) RandomString(n int) string {

	s, r := make([]rune, n), []rune(randomStringSource)

	for i := range s {

		p, _ := rand.Prime(rand.Reader, len(r))
		x, y := p.Uint64(), uint64(len(r))

		s[i] = r[x%y]
	}

	return string(s)
}

// UploadedFile is a struct that stores data regarding an uploaded file

type UploadedFile struct {
	NewFileName      string
	OriginalFileName string
	FileSize         int64
}

// UploadOneFile is just a convenience method that calls UploadFiles, but expects and takes only one file
func (tool *Tools) UploadOneFile(req *http.Request, uploadDir string, rename ...bool) (*UploadedFile, error) {
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}

	files, err := tool.UploadFiles(req, uploadDir, renameFile)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	return files[0], nil

}

/*
UploadFiles takes in a post request and filename and stores

- multiple files may be uploaded
*/
func (tool *Tools) UploadFiles(req *http.Request, uploadDir string, rename ...bool) ([]*UploadedFile, error) {

	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}

	var uploadedFiles []*UploadedFile //* return value

	if tool.MaxFileSize == 0 {
		tool.MaxFileSize = 1024 * 1024 * 1024 //* ~ gigabtye

	}

	// create directory
	err := tool.CreateDirIfNotExist(uploadDir)
	if err != nil {
		return nil, err
	}

	/* func (*http.Request).ParseMultipartForm(maxMemory int64) error
	ParseMultipartForm parses a request body as multipart/form-data. The whole request body is parsed and up to a total of maxMemory bytes of its file parts are stored in memory, with the remainder stored on disk in temporary files.*/
	err = req.ParseMultipartForm(int64(tool.MaxFileSize))
	if err != nil {
		return nil, errors.New("the uploaded file is too large")
	}

	for _, fHeaders := range req.MultipartForm.File {
		for _, hdr := range fHeaders {
			// loop in anon func
			uploadedFiles, err = func(uploadedFiles []*UploadedFile) ([]*UploadedFile, error) {
				var uploadedFile UploadedFile
				infile, err := hdr.Open()

				if err != nil {
					return nil, err
				}

				defer infile.Close()

				buff := make([]byte, 512)
				_, err = infile.Read(buff)
				if err != nil {
					return nil, err
				}

				// check to see if the file type is permitted
				// .: avoid executables, php script, or pearl script
				allowed := false // default assumption
				fileType := http.DetectContentType(buff)
				// allowedTypes := []string{"image/jpeg", "image/png", "image/gif"}

				if len(tool.AllowedFileTypes) > 0 {
					for _, x := range tool.AllowedFileTypes {
						if strings.EqualFold(fileType, x) {
							allowed = true
						}
					}
				} else {
					allowed = true
				}

				if !allowed {
					return nil, errors.New("uploaded file type is not permitted")
				}

				// already read the first 512 bytes of file
				// therefore, must go back to the beginning of
				// the file
				_, err = infile.Seek(0, 0)
				if err != nil {
					return nil, err
				}

				// now deal with accepted file
				// renaming

				if renameFile {
					uploadedFile.NewFileName = fmt.Sprintf("%s%s", tool.RandomString(25), filepath.Ext(hdr.Filename))
				} else {
					uploadedFile.NewFileName = hdr.Filename
				}

				// ? this is odd
				uploadedFile.OriginalFileName = hdr.Filename

				//
				var outfile *os.File
				defer outfile.Close()

				if outfile, err = os.Create(filepath.Join(uploadDir, uploadedFile.NewFileName)); err != nil {
					return nil, err
				} else {
					fileSize, err := io.Copy(outfile, infile)
					if err != nil {
						return nil, err
					}

					uploadedFile.FileSize = fileSize

				}

				uploadedFiles = append(uploadedFiles, &uploadedFile)

				return uploadedFiles, nil

			}(uploadedFiles)

			if err != nil {
				return uploadedFiles, err
			}
		}
	}

	return uploadedFiles, nil

}

// CreateDirIfNotExist creates a directory and necessary parents if the dir does not yet exist
func (tool *Tools) CreateDirIfNotExist(path string) error {
	const mode = 0755

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, mode)

		if err != nil {
			return err
		}
	}

	return nil
}

// Slugify - a simple way to create a slug from a string
func (tool *Tools) Slugify(s string) (string, error) {
	if s == "" {
		return "", errors.New("empty string not permitted")
	}

	// use regex library
	// [^starts with a - z \digits also]
	var re = regexp.MustCompile(`[^a-z\d]+`)
	slug := strings.Trim(re.ReplaceAllString(strings.ToLower(s), "-"), "-")

	if len(slug) == 0 {
		return "", errors.New("after removing characters, slug is zero length")
	}

	return slug, nil
}

// DownloadStaticFile - downloads a file and forces browser to avoid display of file in browser window by setting content disposition and allows specification of the display name
func (tool *Tools) DownloadStaticFile(writer http.ResponseWriter,
	request *http.Request, givenPath, file, displayName string) {

	filePath := path.Join(givenPath, file)

	writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", displayName))

	http.ServeFile(writer, request, filePath)

}

// JSONResponse is the type for sending JSON around
type JSONResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ReadJSON tries to read the body of a request and converts from json to a go data variable
func (tool *Tools) ReadJSON(writer http.ResponseWriter, request *http.Request, data any) error {

	maxBytes := 1024 * 1024 // one mega
	if tool.MaxJSONSize != 0 {
		maxBytes = tool.MaxJSONSize
	}

	// read in the body
	request.Body = http.MaxBytesReader(writer, request.Body, int64(maxBytes))

	// decode it
	decoded := json.NewDecoder(request.Body)

	if !tool.AllowUnknownFields {
		decoded.DisallowUnknownFields()
	}

	err := decoded.Decode(data)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		case err.Error() == "http: request body too large":
			return fmt.Errorf("body must not be larger than %d bytes", maxBytes)

		case errors.As(err, &invalidUnmarshalError):
			return fmt.Errorf("error unmarshalling JSON: %s", err.Error())

		default:
			return err
		}
	}

	err = decoded.Decode(&struct{}{})

	if err != io.EOF {
		return errors.New("body must contain only one JSON value")
	}

	return nil
}

// WriteJSON takes a response status code and arbitrary data and writes json to the client
func (tool *Tools) WriteJSON(responseWriter http.ResponseWriter, status int, data any, headers ...http.Header) error {

	out, err := json.Marshal(data)

	if err != nil {
		return err
	}

	if len(headers) > 0 {
		for key, value := range headers[0] {
			responseWriter.Header()[key] = value
		}
	}

	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(status)
	_, err = responseWriter.Write(out)
	if err != nil {
		return err
	}

	return nil
}

// ErrorJSON takes an error and optionally a status code and generates and sends a JSON error message
func (tool *Tools) ErrorJSON(responseWriter http.ResponseWriter, err error, status ...int) error {

	statusCode := http.StatusBadRequest

	if len(status) > 0 {
		statusCode = status[0]
	}

	// create payload to send back
	var payload JSONResponse
	payload.Error = true
	payload.Message = err.Error()

	// return what to write to
	return tool.WriteJSON(responseWriter, statusCode, payload)

}
