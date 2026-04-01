package pkg

import (
	"crypto/rand"
	"math/big"
)

func GenerateUsername() (string, error) {
	adjIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(Adjectives))))
	if err != nil {
		return "", err
	}

	nounIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(Nouns))))
	if err != nil {
		return "", err
	}

	return Adjectives[adjIdx.Int64()] + "-" + Nouns[nounIdx.Int64()], nil
}
