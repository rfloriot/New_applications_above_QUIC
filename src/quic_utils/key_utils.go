package quic_utils

import (
	"crypto/rsa"
	"encoding/asn1"
	"crypto/x509"
	"crypto/tls"
	"math/big"
	"encoding/pem"
	"bytes"
	"os"
	"io/ioutil"
	"errors"
	"crypto/rand"
	"fmt"
)

// ==================== KEY ENCODING-DECODING ====================

// asn1 encode a public key file
func EncodePublicKey(key *rsa.PublicKey) ([]byte, error) {
	b, err := asn1.Marshal(*key)
	return b, err
}

// asn1 decode a public key file
func DecodePublicKey(b []byte) (*rsa.PublicKey, error) {
	res := rsa.PublicKey{}
	_, err := asn1.Unmarshal(b, &res)
	return &res, err
}

// DER encode a private key
func EncodePrivateKey(key *rsa.PrivateKey) ([]byte){
	return x509.MarshalPKCS1PrivateKey(key)
}

// DER decode a private key
func DecodePrivateKey(der[] byte) (*rsa.PrivateKey, error){
	return x509.ParsePKCS1PrivateKey(der)
}

// compare two keys (return true if equals)
func ComparePublicKeys(key1 *rsa.PublicKey, key2 *rsa.PublicKey) bool {
	return key1.E == key2.E && bytes.Compare(key1.N.Bytes(),key2.N.Bytes()) == 0
}


// ==================== KEY FILE ENCODING-DECODING ====================

// extract content (DER or ASN1) from a PEM file
func ReadPEM(pemFilename string) ([]byte, error){
	f, _ := os.Open(pemFilename)
	certPEM, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to read file '%s'", pemFilename))
	}
	f.Close()

	pemData, _ := pem.Decode(certPEM)
	derData := pemData.Bytes
	return derData, nil
}

// write byte data in a PEM file
func WritePEM(pemFilename string, dataDescription string, rawdata []byte) error {
	f, err := os.Create(pemFilename)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to open PEM file '%s'", pemFilename))
	}

	if pem.Encode(f, &pem.Block{Type: dataDescription, Bytes: rawdata}); err != nil {
		return errors.New(fmt.Sprintf("failed to write data in PEM file '%s'", pemFilename))
	}

	f.Close()
	return nil
}

// extract public key from a asn1 .pub file
func ExtractPublicKey(publicKeyFile string) (*rsa.PublicKey, error) {
	asnData, err := ReadPEM(publicKeyFile)

	if  err == nil {
		return DecodePublicKey(asnData)
	} else {
		return nil, err
	}
}

// extract private key from a pem file
func ExtractPrivateKey(privateKeyFile string) (*rsa.PrivateKey, error) {
	if derData, err := ReadPEM(privateKeyFile); err == nil {
		return DecodePrivateKey(derData)
	} else {
		return nil, err
	}
}

// make a certificate from a pair (public, private) keys
func MakeCertificate(publicKey *rsa.PublicKey, privateKey *rsa.PrivateKey) (tls.Certificate, error){
	template := x509.Certificate{SerialNumber: big.NewInt(1)}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey, privateKey)
	Check(err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: EncodePrivateKey(privateKey)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	return tls.X509KeyPair(certPEM, keyPEM)
}
