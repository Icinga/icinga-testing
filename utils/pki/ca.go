package pki

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
	"math/big"
	"os"
	"time"
)

type CertificateWithKey struct {
	Key               *rsa.PrivateKey
	Certificate       *x509.Certificate
	SignedCertificate []byte
	Request           *x509.CertificateRequest
	SignedRequest     []byte
}

func (c *CertificateWithKey) CertificateToPem() []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: c.SignedCertificate,
	})
}

func (c *CertificateWithKey) KeyToPem() []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(c.Key),
	})
}

func (c *CertificateWithKey) RequestToPem() []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: c.SignedRequest,
	})
}

func (c *CertificateWithKey) CertificateToFile(filename string, perm os.FileMode) error {
	return ioutil.WriteFile(filename, c.CertificateToPem(), perm)
}

func (c *CertificateWithKey) KeyToFile(filename string, perm os.FileMode) error {
	return ioutil.WriteFile(filename, c.KeyToPem(), perm)
}

func (c *CertificateWithKey) RequestToFile(filename string, perm os.FileMode) error {
	return ioutil.WriteFile(filename, c.RequestToPem(), perm)
}

type CA struct {
	CertificateWithKey
}

func NewCA() (*CA, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}
	certificate := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "Icinga CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(100, 0, 0),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	key, err := GenerateRsaKey()
	if err != nil {
		return nil, err
	}

	signedCertificate, err := x509.CreateCertificate(rand.Reader, certificate, certificate, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	certificate, err = x509.ParseCertificate(signedCertificate)
	if err != nil {
		return nil, err
	}

	return &CA{CertificateWithKey{
		Key:               key,
		Certificate:       certificate,
		SignedCertificate: signedCertificate,
	}}, nil
}

func MustNewCA() *CA {
	ca, err := NewCA()
	if err != nil {
		panic(err)
	}
	return ca
}

func (c *CA) NewCertificate(subject string) (*CertificateWithKey, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}
	certificate := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: subject,
		},
		DNSNames:              []string{subject},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(100, 0, 0),
		IsCA:                  false,
		BasicConstraintsValid: true,
	}

	key, err := GenerateRsaKey()
	if err != nil {
		return nil, err
	}

	request := &x509.CertificateRequest{
		SignatureAlgorithm: x509.SHA256WithRSA,
		Subject:            certificate.Subject,
		DNSNames:           certificate.DNSNames,
	}
	signedRequest, err := x509.CreateCertificateRequest(rand.Reader, request, key)
	if err != nil {
		return nil, err
	}

	signedCertificate, err := x509.CreateCertificate(rand.Reader, certificate, c.Certificate, &key.PublicKey, c.Key)
	if err != nil {
		return nil, err
	}

	certificate, err = x509.ParseCertificate(signedCertificate)
	if err != nil {
		return nil, err
	}

	return &CertificateWithKey{
		Key:               key,
		Certificate:       certificate,
		SignedCertificate: signedCertificate,
		Request:           request,
		SignedRequest:     signedRequest,
	}, nil
}

func (c *CA) MustNewCertificate(subject string) *CertificateWithKey {
	certificate, err := c.NewCertificate(subject)
	if err != nil {
		panic(err)
	}
	return certificate
}
