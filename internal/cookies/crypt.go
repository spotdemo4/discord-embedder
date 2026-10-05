package cookies

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
)

// version prefixes encrypted cookie files so the format can change later.
const version byte = 1

var errDecrypt = errors.New("could not decrypt saved cookies")

// encrypt seals data with AES-256-GCM, binding it to userID so files cannot be swapped between users.
func encrypt(key []byte, userID string, data []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}

	out := make([]byte, 1+gcm.NonceSize(), 1+gcm.NonceSize()+len(data)+gcm.Overhead())
	out[0] = version
	nonce := out[1:]
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(out, nonce, data, []byte(userID)), nil
}

func decrypt(key []byte, userID string, data []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}

	if len(data) < 1+gcm.NonceSize() || data[0] != version {
		return nil, errDecrypt
	}
	nonce := data[1 : 1+gcm.NonceSize()]
	plain, err := gcm.Open(nil, nonce, data[1+gcm.NonceSize():], []byte(userID))
	if err != nil {
		return nil, errors.Join(errDecrypt, err)
	}

	return plain, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return cipher.NewGCM(block)
}
