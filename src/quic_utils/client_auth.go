package quic_utils

import (
	"github.com/lucas-clemente/quic-go"
	"crypto/rand"
	"crypto/rsa"
	"encoding/binary"
	"errors"
)

const (
	debug         = false
	nonceByteSize = 16
)

// =========================== client public key exchange =======================================

// merge two []byte
func merge(a []byte, b []byte) []byte {
	return append(a, b...)
}

// generate a nonce of "nonceByteSize"
func genNonce() ([]byte, error) {
	nonce := make([]byte, nonceByteSize, nonceByteSize)
	_, err := rand.Read(nonce)
	return nonce, err
}

// read at most n bytes from stream
func readN(stream quic.Stream, n int) ([]byte, error) {
	buf := make([]byte, n)
	readData, err := stream.Read(buf)
	return buf[:readData], err
}

// encode int to []byte
func encodeInt(val int) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(val))
	return buf
}

// decode []byte to int
func decodeInt(bytes []byte) int {
	return int(binary.BigEndian.Uint32(bytes))
}

// === public functions

type nonceAnswer struct {
	rb        []byte
	keySize   []byte
	key       []byte
	signatureSize []byte
	signature []byte
}

// read a signed nonce response from a stream
func (ans *nonceAnswer) read(stream quic.Stream) (error) {
	rb, err := readN(stream, nonceByteSize)
	ans.rb = rb
	if err != nil {
		return err
	}

	keySize, err := readN(stream, 4)
	ans.keySize = keySize
	if err != nil {
		return err
	}

	key, err := readN(stream, decodeInt(ans.keySize))
	ans.key = key
	if err != nil {
		return err
	}

	signatureSize, err := readN(stream, 4)
	ans.signatureSize = signatureSize
	if err != nil {
		return err
	}

	signature, err := readN(stream, decodeInt(signatureSize))
	ans.signature = signature
	if err != nil {
		return err
	}

	return nil
}

// write a signed nonce response to a stream
func (ans *nonceAnswer) write(session quic.Session, stream quic.Stream) (error) {
	_, err := stream.Write(ans.rb)
	if err != nil {
		return err
	}
	_, err = stream.Write(ans.keySize)
	if err != nil {
		return err
	}
	_, err = stream.Write(ans.key)
	if err != nil {
		return err
	}
	_, err = stream.Write(ans.signatureSize)
	if err != nil {
		return err
	}
	_, err = stream.Write(ans.signature)
	if err != nil {
		return err
	}

	return nil
}

func sign(connectionId uint64, ra []byte, rb[]byte, key *rsa.PrivateKey) ([]byte, error) {
	connectionIdBytes := make([]byte, 8,8)
	binary.BigEndian.PutUint64(connectionIdBytes, connectionId)

	signedContent := merge(ra, rb)
	signedContent = merge(signedContent, connectionIdBytes)

	if signature, err := rsa.SignPKCS1v15(rand.Reader, key, 0, signedContent); err == nil {
		return signature, nil
	} else {
		return nil, errors.New("failed to sign content")
	}
}

func (ans *nonceAnswer) verify(ra []byte) (error) {
	signedContent := merge(ra, ans.rb)
	key, err := DecodePublicKey(ans.key)
	if err != nil {
		return err
	}

	return rsa.VerifyPKCS1v15(key, 0, signedContent, ans.signature)
}

// (client side) accept nonce, answer other nonce, public key + signature
func ServeClientPublicKey(session quic.Session, stream quic.Stream, privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey) error {
	Logf("[client authentication] serve client key ..")

	stream.Write([] byte{0}) // FIXME: dummy data to unlock stream

	// read nonce
	ra, err := readN(stream, nonceByteSize)
	if err != nil {
		return err
	}

	// generate answer
	answer := nonceAnswer{}
	rb, err := genNonce()
	if err == nil {
		answer.rb = rb
	} else {
		return err
	}

	answer.key, err = EncodePublicKey(publicKey)
	if err != nil {
		return err
	}
	answer.keySize = encodeInt(len(answer.key))
	answer.signature, err = sign(session.AddedForThesis_getConnectionId(), ra, answer.rb, privateKey)
	if err != nil {
		return err
	}
	answer.signatureSize = encodeInt(len(answer.signature))

	// send
	if err := answer.write(session, stream); err != nil {
		return err
	}

	return nil
}

// (server side): send nonce, accept answer and check signature
func AskClientPublicKey(session quic.Session, stream quic.Stream) (*rsa.PublicKey, error) {
	Logf("[client authentication] ask client key ..")
	readN(stream, 1) // FIXME: dummy read to avoid blocking

	// send nonce
	ra, err := genNonce()
	if err != nil {
		return nil, err
	}

	if _, err := stream.Write(ra); err != nil {
		return nil, err
	}

	// get answer
	answer := nonceAnswer{}
	if err = answer.read(stream); err != nil {
		return nil, err
	}

	// verify
	key, err := DecodePublicKey(answer.key)
	if err != nil {
		return nil, err
	}

	connectionIDBytes := make([]byte, 8, 8)
	binary.BigEndian.PutUint64(connectionIDBytes, session.AddedForThesis_getConnectionId())

	signedContent := merge(ra, answer.rb)
	signedContent = merge(signedContent, connectionIDBytes)

	return key, rsa.VerifyPKCS1v15(key, 0, signedContent, answer.signature)
}