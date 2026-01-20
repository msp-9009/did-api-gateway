package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/mr-tron/base58"
)

var ed25519Prefix = []byte{0xed, 0x01}

func GenerateEd25519Key() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	return pub, priv, err
}

func EncodeDidKey(pub ed25519.PublicKey) string {
	buf := append([]byte{}, ed25519Prefix...)
	buf = append(buf, pub...)
	return "did:key:z" + base58.Encode(buf)
}

func DecodeDidKey(did string) (ed25519.PublicKey, error) {
	if !strings.HasPrefix(did, "did:key:z") {
		return nil, errors.New("unsupported DID method")
	}
	enc := strings.TrimPrefix(did, "did:key:z")
	raw, err := base58.Decode(enc)
	if err != nil {
		return nil, err
	}
	if len(raw) < len(ed25519Prefix)+ed25519.PublicKeySize {
		return nil, errors.New("invalid did:key length")
	}
	if raw[0] != ed25519Prefix[0] || raw[1] != ed25519Prefix[1] {
		return nil, errors.New("invalid did:key prefix")
	}
	pub := raw[len(ed25519Prefix):]
	if len(pub) != ed25519.PublicKeySize {
		return nil, errors.New("invalid public key size")
	}
	return ed25519.PublicKey(pub), nil
}

func EncodePrivateKey(priv ed25519.PrivateKey) string {
	return base64.RawURLEncoding.EncodeToString(priv)
}

func DecodePrivateKey(enc string) (ed25519.PrivateKey, error) {
	raw, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil {
		return nil, err
	}
	if len(raw) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid private key size")
	}
	return ed25519.PrivateKey(raw), nil
}

func EncodePublicKey(pub ed25519.PublicKey) string {
	return base64.RawURLEncoding.EncodeToString(pub)
}

func DecodePublicKey(enc string) (ed25519.PublicKey, error) {
	raw, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil {
		return nil, err
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, errors.New("invalid public key size")
	}
	return ed25519.PublicKey(raw), nil
}
