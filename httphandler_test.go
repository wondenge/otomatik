package otomatik

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHTTPChallengeHandlerNoOp(t *testing.T) {
	am := &ACMEManager{CA: "https://example.com/acme/directory"}
	testConfig := &Config{
		Issuer:    am,
		Storage:   &FileStorage{Path: "./_testdata_tmp"},
		certCache: new(Cache),
	}
	am.config = testConfig

	testStorageDir := testConfig.Storage.(*FileStorage).Path
	defer func() {
		err := os.RemoveAll(testStorageDir)
		if err != nil {
			t.Fatalf("Could not remove temporary storage directory (%s): %v", testStorageDir, err)
		}
	}()

	// try base paths and host names that aren't
	// handled by this handler
	for _, url := range []string{
		"http://localhost/",
		"http://localhost/foo.html",
		"http://localhost/.git",
		"http://localhost/.well-known/",
		"http://localhost/.well-known/acme-challenging",
		"http://other/.well-known/acme-challenge/foo",
	} {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			t.Fatalf("Could not craft request, got error: %v", err)
		}
		rw := httptest.NewRecorder()
		if am.HandleHTTPChallenge(rw, req) {
			t.Errorf("Got true with this URL, but shouldn't have: %s", url)
		}
	}
}