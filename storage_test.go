package otomatik

import (
	"path"
	"testing"
)

func TestPrefixAndKeyBuilders(t *testing.T) {
	am := &ACMEManager{CA: "https://example.com/acme-ca/directory"}

	base := path.Join("certificates", "example.com-acme-ca-directory")

	for i, testcase := range []struct {
		in, folder, certFile, keyFile, metaFile string
	}{
		{
			in:       "example.com",
			folder:   path.Join(base, "example.com"),
			certFile: path.Join(base, "example.com", "example.com.crt"),
			keyFile:  path.Join(base, "example.com", "example.com.key"),
			metaFile: path.Join(base, "example.com", "example.com.json"),
		},
		{
			in:       "*.example.com",
			folder:   path.Join(base, "wildcard_.example.com"),
			certFile: path.Join(base, "wildcard_.example.com", "wildcard_.example.com.crt"),
			keyFile:  path.Join(base, "wildcard_.example.com", "wildcard_.example.com.key"),
			metaFile: path.Join(base, "wildcard_.example.com", "wildcard_.example.com.json"),
		},
		{
			// prevent directory traversal! very important, esp. with on-demand TLS
			// see issue #2092
			in:       "a/../../../foo",
			folder:   path.Join(base, "afoo"),
			certFile: path.Join(base, "afoo", "afoo.crt"),
			keyFile:  path.Join(base, "afoo", "afoo.key"),
			metaFile: path.Join(base, "afoo", "afoo.json"),
		},
		{
			in:       "b\\..\\..\\..\\foo",
			folder:   path.Join(base, "bfoo"),
			certFile: path.Join(base, "bfoo", "bfoo.crt"),
			keyFile:  path.Join(base, "bfoo", "bfoo.key"),
			metaFile: path.Join(base, "bfoo", "bfoo.json"),
		},
		{
			in:       "c/foo",
			folder:   path.Join(base, "cfoo"),
			certFile: path.Join(base, "cfoo", "cfoo.crt"),
			keyFile:  path.Join(base, "cfoo", "cfoo.key"),
			metaFile: path.Join(base, "cfoo", "cfoo.json"),
		},
	} {
		if actual := StorageKeys.SiteCert(am.IssuerKey(), testcase.in); actual != testcase.certFile {
			t.Errorf("Test %d: site cert file: Expected '%s' but got '%s'", i, testcase.certFile, actual)
		}
		if actual := StorageKeys.SitePrivateKey(am.IssuerKey(), testcase.in); actual != testcase.keyFile {
			t.Errorf("Test %d: site key file: Expected '%s' but got '%s'", i, testcase.keyFile, actual)
		}
		if actual := StorageKeys.SiteMeta(am.IssuerKey(), testcase.in); actual != testcase.metaFile {
			t.Errorf("Test %d: site meta file: Expected '%s' but got '%s'", i, testcase.metaFile, actual)
		}
	}
}