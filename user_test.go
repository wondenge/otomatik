package otomatik

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"github.com/go-acme/lego/v3/registration"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUser(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatalf("Could not generate test private key: %v", err)
	}
	u := user{
		Email:        "me@mine.com",
		Registration: new(registration.Resource),
		key:          privateKey,
	}

	if expected, actual := "me@mine.com", u.GetEmail(); actual != expected {
		t.Errorf("Expected email '%s' but got '%s'", expected, actual)
	}
	if u.GetRegistration() == nil {
		t.Error("Expected a registration resource, but got nil")
	}
	if expected, actual := privateKey, u.GetPrivateKey(); actual != expected {
		t.Errorf("Expected the private key at address %p but got one at %p instead ", expected, actual)
	}
}
func TestNewUser(t *testing.T) {
	am := &ACMEManager{CA: dummyCA}
	testConfig := &Config{
		Issuer:    am,
		Storage:   &FileStorage{Path: "./_testdata_tmp"},
		certCache: new(Cache),
	}
	am.config = testConfig

	email := "me@foobar.com"
	user, err := am.newUser(email)
	if err != nil {
		t.Fatalf("Error creating user: %v", err)
	}
	if user.key == nil {
		t.Error("Private key is nil")
	}
	if user.Email != email {
		t.Errorf("Expected email to be %s, but was %s", email, user.Email)
	}
	if user.Registration != nil {
		t.Error("New user already has a registration resource; it shouldn't")
	}
}

func TestSaveUser(t *testing.T) {
	am := &ACMEManager{CA: dummyCA}
	testConfig := &Config{
		Issuer:    am,
		Storage:   &FileStorage{Path: "./_testdata1_tmp"},
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

	email := "me@foobar.com"
	user, err := am.newUser(email)
	if err != nil {
		t.Fatalf("Error creating user: %v", err)
	}

	err = am.saveUser(am.CA, user)
	if err != nil {
		t.Fatalf("Error saving user: %v", err)
	}
	_, err = am.getUser(am.CA, email)
	if err != nil {
		t.Errorf("Cannot access user data, error: %v", err)
	}
}

func TestGetUserDoesNotAlreadyExist(t *testing.T) {
	am := &ACMEManager{CA: dummyCA}
	testConfig := &Config{
		Issuer:    am,
		Storage:   &FileStorage{Path: "./_testdata_tmp"},
		certCache: new(Cache),
	}
	am.config = testConfig

	user, err := am.getUser(am.CA, "user_does_not_exist@foobar.com")
	if err != nil {
		t.Fatalf("Error getting user: %v", err)
	}

	if user.key == nil {
		t.Error("Expected user to have a private key, but it was nil")
	}
}

func TestGetUserAlreadyExists(t *testing.T) {
	am := &ACMEManager{CA: dummyCA}
	testConfig := &Config{
		Issuer:    am,
		Storage:   &FileStorage{Path: "./_testdata2_tmp"},
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

	email := "me@foobar.com"

	// Set up test
	user, err := am.newUser(email)
	if err != nil {
		t.Fatalf("Error creating user: %v", err)
	}
	err = am.saveUser(am.CA, user)
	if err != nil {
		t.Fatalf("Error saving user: %v", err)
	}

	// Expect to load user from disk
	user2, err := am.getUser(am.CA, email)
	if err != nil {
		t.Fatalf("Error getting user: %v", err)
	}

	// Assert keys are the same
	if !privateKeysSame(user.key, user2.key) {
		t.Error("Expected private key to be the same after loading, but it wasn't")
	}

	// Assert emails are the same
	if user.Email != user2.Email {
		t.Errorf("Expected emails to be equal, but was '%s' before and '%s' after loading", user.Email, user2.Email)
	}
}

func TestGetEmailFromPackageDefault(t *testing.T) {
	DefaultACME.Email = "tEsT2@foo.com"
	defer func() {
		DefaultACME.Email = ""
	}()

	am := &ACMEManager{CA: dummyCA}
	testConfig := &Config{
		Issuer:    am,
		Storage:   &FileStorage{Path: "./_testdata2_tmp"},
		certCache: new(Cache),
	}
	am.config = testConfig

	err := am.getEmail(true)
	if err != nil {
		t.Fatalf("getEmail error: %v", err)
	}
	lowerEmail := strings.ToLower(DefaultACME.Email)
	if am.Email != lowerEmail {
		t.Errorf("Did not get correct email from memory; expected '%s' but got '%s'", lowerEmail, am.Email)
	}
}

func TestGetEmailFromUserInput(t *testing.T) {
	am := &ACMEManager{CA: dummyCA}
	testConfig := &Config{
		Issuer:    am,
		Storage:   &FileStorage{Path: "./_testdata3_tmp"},
		certCache: new(Cache),
	}
	am.config = testConfig

	// let's not clutter up the output
	origStdout := os.Stdout
	os.Stdout = nil
	agreementTestURL = "(none - testing)"
	defer func() {
		os.Stdout = origStdout
		agreementTestURL = ""
	}()

	email := "test3@foo.com"
	stdin = bytes.NewBufferString(email + "\n")
	err := am.getEmail(true)
	if err != nil {
		t.Fatalf("getEmail error: %v", err)
	}
	if am.Email != email {
		t.Errorf("Did not get correct email from user input prompt; expected '%s' but got '%s'", email, am.Email)
	}
	if !am.Agreed {
		t.Error("Expect Config.Agreed to be true, but got false")
	}
}

func TestGetEmailFromRecent(t *testing.T) {
	am := &ACMEManager{CA: dummyCA}
	testConfig := &Config{
		Issuer:    am,
		Storage:   &FileStorage{Path: "./_testdata4_tmp"},
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

	DefaultACME.Email = ""

	for i, eml := range []string{
		"test4-1@foo.com",
		"test4-2@foo.com",
		"TEST4-3@foo.com", // test case insensitivity
	} {
		u, err := am.newUser(eml)
		if err != nil {
			t.Fatalf("Error creating user %d: %v", i, err)
		}
		err = am.saveUser(am.CA, u)
		if err != nil {
			t.Fatalf("Error saving user %d: %v", i, err)
		}

		// Change modified time so they're all different and the test becomes more deterministic
		fs := testConfig.Storage.(*FileStorage)
		userFolder := filepath.Join(fs.Path, am.storageKeyUserPrefix(am.CA, eml))
		f, err := os.Stat(userFolder)
		if err != nil {
			t.Fatalf("Could not access user folder for '%s': %v", eml, err)
		}
		chTime := f.ModTime().Add(time.Duration(i) * time.Hour) // 1 second isn't always enough spacing!
		if err := os.Chtimes(userFolder, chTime, chTime); err != nil {
			t.Fatalf("Could not change user folder mod time for '%s': %v", eml, err)
		}
	}
	err := am.getEmail(true)
	if err != nil {
		t.Fatalf("getEmail error: %v", err)
	}
	if am.Email != "test4-3@foo.com" {
		t.Errorf("Did not get correct email from storage; expected '%s' but got '%s'", "test4-3@foo.com", am.Email)
	}
}