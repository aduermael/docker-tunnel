package main

import (
	"io/ioutil"
	"os/user"
	"path/filepath"

	"github.com/aduermael/crypto/ssh"
)

// authMethodPublicKeys returns an ssh.PublicKeys authentication
// method using private key path.
// If privateKeyPath is empty, default location used: ~/.ssh/id_rsa
func authMethodPublicKeys(privateKeyPath string) (ssh.AuthMethod, error) {

	if privateKeyPath == "" {
		usr, err := user.Current()
		if err != nil {
			return nil, err
		}
		privateKeyPath = filepath.Join(usr.HomeDir, ".ssh", "id_rsa")
	}

	key, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	return ssh.PublicKeys(signer), nil
}
