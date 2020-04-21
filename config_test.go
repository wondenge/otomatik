package otomatik

import (
	"github.com/go-acme/lego/v3/certificate"
	"os"
	"reflect"
	"testing"
)

func TestSaveCertResource(t *testing.T) {
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

	domain := "example.com"
	certContents := "certificate"
	keyContents := "private key"

	cert := CertificateResource{
		SANs:           []string{domain},
		PrivateKeyPEM:  []byte(keyContents),
		CertificatePEM: []byte(certContents),
		IssuerData: &certificate.Resource{
			Domain:        domain,
			CertURL:       "https://example.com/cert",
			CertStableURL: "https://example.com/cert/stable",
		},
	}

	err := testConfig.saveCertResource(cert)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// the result of our test will be a map, since we have
	// no choice but to decode it into an interface
	cert.IssuerData = map[string]interface{}{
		"domain":        domain,
		"certUrl":       "https://example.com/cert",
		"certStableUrl": "https://example.com/cert/stable",
	}

	siteData, err := testConfig.loadCertResource(domain)
	if err != nil {
		t.Fatalf("Expected no error reading site, got: %v", err)
	}
	if !reflect.DeepEqual(cert, siteData) {
		t.Errorf("Expected '%+v' to match '%+v'", cert, siteData)
	}
}