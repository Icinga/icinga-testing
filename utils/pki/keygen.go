package pki

import (
	"crypto/rand"
	"crypto/rsa"
	"runtime"
	"sync"
)

type rsaGenerateKeyResult struct {
	key *rsa.PrivateKey
	err error
}

var setupKeygenPoolOnce sync.Once
var rsaKeys chan *rsaGenerateKeyResult

func GenerateRsaKey() (*rsa.PrivateKey, error) {
	setupKeygenPoolOnce.Do(func() {
		rsaKeys = make(chan *rsaGenerateKeyResult)
		for i := 0; i < runtime.NumCPU(); i++ {
			go func() {
				for {
					key, err := rsa.GenerateKey(rand.Reader, 2048)
					rsaKeys <- &rsaGenerateKeyResult{
						key: key,
						err: err,
					}
				}
			}()
		}
	})

	result := <-rsaKeys
	return result.key, result.err
}
