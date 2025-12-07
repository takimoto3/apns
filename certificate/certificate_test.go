package certificate_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/takimoto3/apns/certificate"
	pkcs12lib "software.sslmate.com/src/go-pkcs12"
)

// createTestP12 generates a .p12 file (valid or invalid) at a temporary location.
// It returns the file path and a cleanup function.
func createTestP12(t *testing.T, password string, valid bool) (filePath string, cleanup func()) {
	t.Helper()

	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test_apns_*.p12")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	filePath = tmpfile.Name()
	tmpfile.Close() // Close immediately as we will write to it later

	cleanup = func() {
		os.Remove(filePath) // Clean up the temporary file
	}

	if !valid {
		// Create an intentionally invalid .p12 file (e.g., just some random bytes)
		err := os.WriteFile(filePath, []byte("this is not a valid p12 file"), 0600)
		if err != nil {
			cleanup()
			t.Fatalf("Failed to write invalid data to temp file: %v", err)
		}
		return filePath, cleanup
	}

	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to generate RSA private key: %v", err)
	}

	// Create a self-signed X.509 certificate
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Corp"},
			CommonName:   "test.example.com",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // Valid for 1 year
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to create certificate: %v", err)
	}

	certificate, err := x509.ParseCertificate(derBytes)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	// Encode the private key and certificate into a P12 bundle
	// pkcs12lib.Encode expects (rand.Reader, privateKey, certificate, caCerts []*x509.Certificate, password string)
	p12Data, err := pkcs12lib.Encode(rand.Reader, privateKey, certificate, nil, password) // nil for intermediate CAs
	if err != nil {
		cleanup()
		t.Fatalf("Failed to encode PKCS#12 bundle: %v", err)
	}

	// Write the P12 data to the temporary file
	err = os.WriteFile(filePath, p12Data, 0600)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to write PKCS#12 data to temp file: %v", err)
	}

	return filePath, cleanup
}

func TestLoad(t *testing.T) {
	// Test Case 1: P12 file does not exist (still needs to be handled, but path will be dummy)
	t.Run("NonExistentP12File", func(t *testing.T) {
		_, err := certificate.LoadP12File("non_existent.p12", "password")
		if err == nil {
			t.Errorf("LoadP12File expected an error for non-existent file, but got nil")
		}
		if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
			t.Errorf("LoadP12File got unexpected error for non-existent file: %v", err)
		}
	})

	// Test Case 2: Valid P12 file but incorrect password
	t.Run("ValidP12FileAndIncorrectPassword", func(t *testing.T) {
		validP12Path, cleanup := createTestP12(t, "correctPassword", true)
		defer cleanup()

		_, err := certificate.LoadP12File(validP12Path, "incorrectPassword")
		if err == nil {
			t.Errorf("LoadP12File expected an error for incorrect password, but got nil")
		}
		if err != nil && err.Error() != "failed to decode p12 file: pkcs12: decryption password incorrect" {
			t.Errorf("LoadP12File got unexpected error for incorrect password: %v", err)
		}
	})

	// Test Case 3: Valid P12 file with correct password
	t.Run("ValidP12FileAndCorrectPassword", func(t *testing.T) {
		validP12Path, cleanup := createTestP12(t, "correctPassword", true)
		defer cleanup()

		cert, err := certificate.LoadP12File(validP12Path, "correctPassword")
		if err != nil {
			t.Fatalf("LoadP12File failed unexpectedly for valid file and correct password: %v", err)
		}

		if len(cert.Certificate) == 0 {
			t.Errorf("Loaded tls.Certificate is empty (no raw certificate bytes)")
		}
		if cert.PrivateKey == nil {
			t.Errorf("Loaded tls.Certificate has a nil PrivateKey")
		}
		t.Logf("Successfully loaded P12 file from: %s", validP12Path)
	})

	// Test Case 4: Invalid P12 file format
	t.Run("InvalidP12FileFormat", func(t *testing.T) {
		invalidP12Path, cleanup := createTestP12(t, "", false) // Password doesn't matter for invalid format
		defer cleanup()

		_, err := certificate.LoadP12File(invalidP12Path, "password")
		if err == nil {
			t.Errorf("LoadP12File expected an error for invalid file format, but got nil")
		}
		// Expecting pkcs12.DecodeChain to fail for non-p12 data
		if err != nil && !strings.HasPrefix(err.Error(), "failed to decode p12 file:") {
			t.Errorf("LoadP12File got unexpected error for invalid format: %v", err)
		}
	})
}
