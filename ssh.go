package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os/user"
	"path/filepath"

	"github.com/aduermael/crypto/ssh"
	"github.com/howeyc/gopass"
)

const (
	errCannotDecodeEncryptedPrivateKeys = "ssh: cannot decode encrypted private keys"
)

// authMethodPublicKeys returns an ssh.PublicKeys authentication
// method using private key path.
// If privateKeyPath is empty, default location used: ~/.ssh/id_rsa
func authMethodPublicKeys(privateKeyPath, password string) (ssh.AuthMethod, error) {

	if privateKeyPath == "" {
		usr, err := user.Current()
		if err != nil {
			return nil, err
		}
		privateKeyPath = filepath.Join(usr.HomeDir, ".ssh", "id_rsa")
	}

	pemBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, err
	}

	key, err := ssh.ParseRawPrivateKey(pemBytes)
	if err != nil {
		if err.Error() == errCannotDecodeEncryptedPrivateKeys {
			// private key is encrypted, try to decrypt it
			key, err = decryptPrivateKey(pemBytes, []byte(keyPassword))
			if err != nil {
				// prompt user for ssh key password
				fmt.Printf("Enter password for private key (%s): ", privateKeyPath)
				var passwordInput []byte
				passwordInput, err = gopass.GetPasswd()
				if err != nil {
					return nil, err
				}
				key, err = decryptPrivateKey(pemBytes, passwordInput)
				if err != nil {
					return nil, err
				}
			}
		} else {
			return nil, err
		}
	}

	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		return nil, err
	}

	return ssh.PublicKeys(signer), nil
}

// decryptPrivateKey decryps a private key using provided password
func decryptPrivateKey(pemBytes []byte, password []byte) (*rsa.PrivateKey, error) {
	block, rest := pem.Decode(pemBytes)
	if len(rest) > 0 {
		return nil, errors.New("extra data included in key")
	}
	der, err := x509.DecryptPEMBlock(block, password)
	if err != nil {
		return nil, fmt.Errorf("decrypt failed: %v", err)
	}
	key, err := x509.ParsePKCS1PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("decrypt failed: %v", err)
	}
	return key, nil
}
