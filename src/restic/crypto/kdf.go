package crypto

import (
	"crypto/rand"
	"fmt"
	"time"

	sscrypt "github.com/elithrar/simple-scrypt"
	"golang.org/x/crypto/scrypt"
)

const saltLength = 64

// KDFParams are the default parameters used for the key derivation function KDF().
type KDFParams struct {
	N int
	R int
	P int
}

// DefaultKDFParams are the default parameters used for Calibrate and KDF().
var DefaultKDFParams = KDFParams{
	N: sscrypt.DefaultParams.N,
	R: sscrypt.DefaultParams.R,
	P: sscrypt.DefaultParams.P,
}

// Calibrate determines new KDF parameters for the current hardware.
func Calibrate(timeout time.Duration, memory int) (KDFParams, error) {
	defaultParams := sscrypt.Params{
		N:       DefaultKDFParams.N,
		R:       DefaultKDFParams.R,
		P:       DefaultKDFParams.P,
		DKLen:   sscrypt.DefaultParams.DKLen,
		SaltLen: sscrypt.DefaultParams.SaltLen,
	}

	params, err := sscrypt.Calibrate(timeout, memory, defaultParams)
	if err != nil {
		return DefaultKDFParams, err
	}

	return KDFParams{
		N: params.N,
		R: params.R,
		P: params.P,
	}, nil
}

// KDF derives encryption and message authentication keys from the password
// using the supplied parameters N, R and P and the Salt.
func KDF(p KDFParams, salt []byte, password string) (*Key, error) {
	if len(salt) != saltLength {
		return nil, fmt.Errorf("scrypt() called with invalid salt bytes (len %d)", len(salt))
	}

	// make sure we have valid parameters
	params := sscrypt.Params{
		N:       p.N,
		R:       p.R,
		P:       p.P,
		DKLen:   sscrypt.DefaultParams.DKLen,
		SaltLen: len(salt),
	}

	if err := params.Check(); err != nil {
		return nil, err
	}

	derKeys := &Key{}

	keybytes := macKeySize + aesKeySize
	scryptKeys, err := scrypt.Key([]byte(password), salt, p.N, p.R, p.P, keybytes)
	if err != nil {
		return nil, fmt.Errorf("error deriving keys from password: %v", err)
	}

	if len(scryptKeys) != keybytes {
		return nil, fmt.Errorf("invalid numbers of bytes expanded from scrypt(): %d", len(scryptKeys))
	}

	// first 32 byte of scrypt output is the encryption key
	copy(derKeys.Encrypt[:], scryptKeys[:aesKeySize])

	// next 32 byte of scrypt output is the mac key, in the form k||r
	macKeyFromSlice(&derKeys.MAC, scryptKeys[aesKeySize:])

	return derKeys, nil
}

// NewSalt returns new random salt bytes to use with KDF(). If NewSalt returns
// an error, this is a grave situation and the program must abort and terminate.
func NewSalt() ([]byte, error) {
	buf := make([]byte, saltLength)
	n, err := rand.Read(buf)
	if n != saltLength || err != nil {
		panic("unable to read enough random bytes for new salt")
	}

	return buf, nil
}
