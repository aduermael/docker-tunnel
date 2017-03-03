package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/aduermael/crypto/ssh"
	"github.com/spf13/cobra"
)

var (
	// path to private key or empty to use default
	sshIdentityFile = ""
	// open a bash session by default, but different option can be used
	shell = "bash"
	// proxy mode (don't start shell session)
	proxyMode = false
)

func main() {

	rootCmd := &cobra.Command{
		Use:   "docker-tunnel [user@]host",
		Short: "Docker-tunnel connects you to remote Docker hosts using SSH tunnels",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				cmd.Usage()
				return
			}

			sshClient, err := sshConnect(args[0], sshIdentityFile)
			if err != nil {
				log.Fatalln(err)
			}
			defer sshClient.Close()

			if proxyMode {
				ln, err := net.Listen("tcp", ":2375")
				if err != nil {
					log.Fatalln(err)
				}
				fmt.Println("listening on port 2375...")
				for {
					conn, err := ln.Accept()
					if err != nil {
						log.Fatalln(err)
					}
					go handleProxyConnection(conn, sshClient)
				}
				return
			}

			// proxyMode == false
			// open shell session, connected to remote Docker host

			socketPath := tmpSocketPath()

			ln, err := net.Listen("unix", socketPath)
			if err != nil {
				log.Fatalln(err)
			}
			defer os.RemoveAll(socketPath)

			// listen in background
			go func() {
				for {
					conn, err := ln.Accept()
					if err != nil {
						log.Fatalln(err)
					}
					go handleProxyConnection(conn, sshClient)
				}
			}()

			os.Setenv("PS1", "🐳  $ ")
			os.Setenv("DOCKER_HOST", "unix://"+socketPath)

			sh := exec.Command(shell)
			sh.Stdout = os.Stdout
			sh.Stderr = os.Stderr
			sh.Stdin = os.Stdin

			err = sh.Run()
			if err != nil {
				log.Fatalln(err.Error())
			}
		},
	}

	rootCmd.Flags().StringVarP(&sshIdentityFile, "sshid", "i", "", "path to private key")
	rootCmd.Flags().StringVarP(&shell, "shell", "s", "bash", "shell to open session")
	rootCmd.Flags().BoolVarP(&proxyMode, "proxy", "p", false, "proxy mode (don't start shell session)")

	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err.Error())
	}
}

func tmpSocketPath() string {
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	return filepath.Join(os.TempDir(), "docker-"+hex.EncodeToString(randBytes)+".sock")
}

func sshConnect(userAtHost string, privateKeyPath string) (*ssh.Client, error) {
	// root user by default
	user := "root"
	host := ""
	userAndHost := strings.SplitN(userAtHost, "@", 2)
	if len(userAndHost) == 1 {
		host = userAndHost[0]
	} else {
		user = userAndHost[0]
		host = userAndHost[1]
	}

	u, err := url.Parse(host)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("ssh connection can't be established: %s", err))
	}
	if u.Scheme == "" {
		u.Scheme = "tcp"
	}

	authMethod, err := authMethodPublicKeys(privateKeyPath)
	if err != nil {
		log.Fatalln(err)
	}
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{authMethod},
	}

	addr := u.Host + u.Path
	parts := strings.Split(addr, ":")
	lastPart := parts[len(parts)-1]
	_, err = strconv.Atoi(lastPart)
	// port is required, used 22 by default
	if err != nil {
		addr += ":22"
	}

	sshClientConn, err := ssh.Dial(u.Scheme, addr, config)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("ssh connection can't be established: %s", err))
	}
	return sshClientConn, nil
}

func handleProxyConnection(conn net.Conn, sshClient *ssh.Client) {
	err := forward(conn, sshClient, "unix:///var/run/docker.sock")
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't forward connection: %s\n", err.Error())
	}
}

func forward(conn net.Conn, sshClient *ssh.Client, remoteAddr string) error {

	// parse OpenSSH version, 6.7 is the minimum required
	reOpenSSH := regexp.MustCompile("OpenSSH_[.0-9]+")
	reOpenSSHVersion := regexp.MustCompile("[.0-9]+")
	match := reOpenSSH.Find(sshClient.ServerVersion())
	openSSHVersionStr := string(reOpenSSHVersion.Find(match))
	openSSHVersion, err := strconv.ParseFloat(openSSHVersionStr, 64)
	if err != nil {
		return errors.New("can't parse server OpenSSH version")
	}
	if openSSHVersion < 6.7 {
		return errors.New("OpenSSH 6.7 minimum required on server side")
	}

	// remote addr
	u, err := url.Parse(remoteAddr)
	if err != nil {
		return errors.New(fmt.Sprintf("can't parse remote address: %s\n", remoteAddr))
	}

	addr := filepath.Join(u.Host, u.Path)

	sshConn, err := sshClient.Dial(u.Scheme, addr)
	if err != nil {
		return errors.New(fmt.Sprintf("can't connect to %s (from remote)", remoteAddr))
	}

	// Copy conn.Reader to sshConn.Writer
	go func() {
		_, err = io.Copy(sshConn, conn)
		if err != nil {
			if err != io.EOF {
				log.Fatalln(err)
			}
		}
	}()

	// Copy sshConn.Reader to localConn.Writer
	go func() {
		_, err = io.Copy(conn, sshConn)
		if err != nil {
			if err != io.EOF {
				log.Fatalln(err)
			}
		}
	}()

	return nil
}
