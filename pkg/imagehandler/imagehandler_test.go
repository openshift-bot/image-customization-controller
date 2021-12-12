/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package imagehandler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type closer struct {
	io.ReadSeeker
}

func (c closer) Close() error {
	return nil
}

func nopCloser(stream io.ReadSeeker) io.ReadSeekCloser {
	return closer{stream}
}

func TestImageHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/host-xyz-45-uuid", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	imageServer := &imageFileSystem{
		log:     zap.New(zap.UseDevMode(true)),
		isoFile: &baseIso{baseFileData{filename: "dummyfile.iso", size: 12345}},
		baseURL: "http://localhost:8080",
		keys: map[string]string{
			"host-xyz-45-uuid": "host-xyz-45.iso",
		},
		images: map[string]*imageFile{
			"host-xyz-45.iso": {
				name:            "host-xyz-45-uuid",
				size:            12345,
				ignitionContent: []byte("asietonarst"),
				imageReader:     nopCloser(strings.NewReader("aiosetnarsetin")),
			},
		},
		mu: &sync.Mutex{},
	}

	handler := http.FileServer(imageServer.FileSystem())
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected := `aiosetnarsetin`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestNewImageHandler(t *testing.T) {
	handler := NewImageHandler(zap.New(zap.UseDevMode(true)),
		"dummyfile.iso",
		"dummyfile.initramfs",
		"http://base.test:1234")

	ifs := handler.(*imageFileSystem)
	ifs.isoFile.size = 12345
	ifs.initramfsFile.size = 12345

	url1, err := handler.ServeImage("test-name-1", []byte{}, false, false)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	_, err = handler.ServeImage("test-name-2", []byte{}, true, false)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	url1again, err := handler.ServeImage("test-name-1", []byte{}, false, false)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if url1again != url1 {
		t.Errorf("inconsistent URLs for same key: %s %s", url1, url1again)
	}
}

func TestNewImageHandlerStatic(t *testing.T) {
	handler := NewImageHandler(zap.New(zap.UseDevMode(true)),
		"dummyfile.iso",
		"dummyfile.initramfs",
		"http://base.test:1234")

	ifs := handler.(*imageFileSystem)
	ifs.isoFile.size = 12345
	ifs.initramfsFile.size = 12345

	url1, err := handler.ServeImage("test-name-1.iso", []byte{}, false, true)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	url2, err := handler.ServeImage("test-name-2.initramfs", []byte{}, true, true)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	url1again, err := handler.ServeImage("test-name-1.iso", []byte{}, false, true)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	url1Expected := "http://base.test:1234/test-name-1.iso"
	if url1 != url1Expected {
		t.Errorf("unexpected url %s (should be %s)", url1, url1Expected)
	}
	url2Expected := "http://base.test:1234/test-name-2.initramfs"
	if url2 != url2Expected {
		t.Errorf("unexpected url %s (should be %s)", url2, url2Expected)
	}
	if url1again != url1 {
		t.Errorf("inconsistent URLs for same key: %s %s", url1, url1again)
	}
}
