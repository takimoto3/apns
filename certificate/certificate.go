package certificate

import (
	"crypto/tls"
	"fmt"
	"os"

	"software.sslmate.com/src/go-pkcs12"
)

// LoadAPNsCertificateFromP12 loads a tls.Certificate for APNs connection
// from a specified p12 file and password.
//
// p12FilePath: Path to the PKCS#12 file.
// password: Password for the p12 file.
//
// Returns:
//
//	*tls.Certificate: A pointer to tls.Certificate on success.
//	error: Error information if loading fails.
func LoadP12File(path, password string) (*tls.Certificate, error) {
	// Read the p12 file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read p12 file %q: %w", path, err)
	}

	// Decode the p12 data using the go-pkcs12 library.
	// This extracts the private key and certificate (and intermediate CA certificates).
	prikey, cert, caCerts, err := pkcs12.DecodeChain(data, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decode p12 file: %w", err)
	}

	// Create a tls.Certificate using the extracted private key and certificate.
	// The 'Certificate' field of tls.Certificate expects a slice of DER-encoded byte slices.
	// Add the Leaf Certificate (the main certificate used for APNs connection) first.
	tlsCert := tls.Certificate{
		Certificate: [][]byte{cert.Raw},
		PrivateKey:  prikey,
	}

	// Optionally, add the CA certificate chain.
	// For APNs, the Leaf Certificate is usually enough.
	// Add CAs if strict client authentication requires the full chain in the TLS handshake.
	for _, caCert := range caCerts {
		tlsCert.Certificate = append(tlsCert.Certificate, caCert.Raw)
	}

	return &tlsCert, nil
}
