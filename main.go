package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

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
	// verbose mode (debug logs)
	verbose = false
)

func main() {

	rootCmd := &cobra.Command{
		Use:   "docker-tunnel [user@]host",
		Short: "Docker-tunnel connects you to a remote Docker host through an SSH tunnel",
		Run: func(cmd *cobra.Command, args []string) {
			if verbose {
				logLevel = logLevelDebug
			}

			if len(args) != 1 {
				cmd.Usage()
				return
			}

			sshClient, err := sshConnect(args[0], sshIdentityFile)
			if err != nil {
				printFatal(err)
			}
			defer sshClient.Close()

			if proxyMode {
				printDebug("proxy mode")

				ln, err := net.Listen("tcp", ":2375")
				if err != nil {
					printFatal(err)
				}
				print("listening on port 2375...")
				for {
					conn, err := ln.Accept()
					if err != nil {
						printFatal(err)
					}
					go handleProxyConnection(conn, sshClient)
				}
			}

			// proxyMode == false
			// open shell session, connected to remote Docker host
			printDebug("shell mode")

			socketPath := tmpSocketPath()
			printDebug("socket path:", socketPath)

			ln, err := net.Listen("unix", socketPath)
			if err != nil {
				printFatal(err)
			}
			defer os.RemoveAll(socketPath)

			// listen in background
			go func() {
				for {
					conn, err := ln.Accept()
					if err != nil {
						printFatal(err)
					}
					printDebug("handle socket connection")
					go handleProxyConnection(conn, sshClient)
				}
			}()

			os.Setenv("PS1", "🐳  $ ")
			os.Setenv("DOCKER_HOST", "unix://"+socketPath)

			sh := exec.Command(shell)
			sh.Stdout = os.Stdout
			sh.Stderr = os.Stderr
			sh.Stdin = os.Stdin

			_ = sh.Run()
		},
	}

	rootCmd.Flags().StringVarP(&sshIdentityFile, "sshid", "i", "", "path to private key")
	rootCmd.Flags().StringVarP(&shell, "shell", "s", "bash", "shell to open session")
	rootCmd.Flags().BoolVarP(&proxyMode, "proxy", "p", false, "proxy mode (don't start shell session)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose mode (debug logs)")

	if err := rootCmd.Execute(); err != nil {
		printFatal(err.Error())
	}
}

func tmpSocketPath() string {
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	return filepath.Join(os.TempDir(), "docker-"+hex.EncodeToString(randBytes)+".sock")
}

func sshConnect(userAtHost, privateKeyPath string) (*ssh.Client, error) {
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

	printDebug("user:", user)

	u, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("ssh connection can't be established: %s", err)
	}
	if u.Scheme == "" {
		u.Scheme = "tcp"
	}

	authMethod, err := authMethodPublicKeys(privateKeyPath)
	if err != nil {
		printFatal(err)
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

	printDebug("address:", u.Scheme+"://"+addr)

	sshClientConn, err := ssh.Dial(u.Scheme, addr, config)
	if err != nil {
		return nil, fmt.Errorf("ssh connection can't be established: %s", err)
	}

	printDebug("ssh connection established")

	return sshClientConn, nil
}

func handleProxyConnection(conn net.Conn, sshClient *ssh.Client) {
	err := forward(conn, sshClient, "unix:///var/run/docker.sock")
	if err != nil {
		printError(os.Stderr, "can't forward connection:", err.Error())
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
		return fmt.Errorf("can't parse remote address: %s", remoteAddr)
	}

	addr := filepath.Join(u.Host, u.Path)

	sshConn, err := sshClient.Dial(u.Scheme, addr)
	if err != nil {
		return fmt.Errorf("can't connect to %s (from remote)", remoteAddr)
	}

	chan1 := make(chan struct{})
	chan2 := make(chan struct{})
	chan3 := make(chan struct{})
	var o sync.Once
	closeChan2 := func() {
		close(chan2)
	}

	// Copy conn.Reader to sshConn.Writer
	go func() {
		printDebug("copy: read from client conn, write to server conn")
		_, err := io.Copy(sshConn, conn)
		if err != nil {
			printFatal(err)
		}
		close(chan1)

		if c, ok := sshConn.(ssh.Channel); ok {
			if err := c.CloseWrite(); err != nil {
				printError("can't close sshConn writer")
			} else {
				printDebug("closed sshConn writer")
			}

		}

		for {
			printDebug("can't read from client anymore, trying to write...")
			_, err := conn.Write(make([]byte, 0))
			if err != nil {
				printDebug("can't write, closing both connections")
				o.Do(closeChan2)
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	// Copy sshConn.Reader to localConn.Writer
	go func() {
		printDebug("copy: read from server conn, write to client conn")
		_, err := io.Copy(conn, sshConn)
		if err != nil {
			printFatal(err)
		}
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			if err := tcpConn.CloseWrite(); err != nil {
				printError("can't close TCPconn writer")
			} else {
				printDebug("closed TCPconn writer")
			}
		} else if unixConn, ok := conn.(*net.UnixConn); ok {
			if err := unixConn.CloseWrite(); err != nil {
				printError("can't close UnixConn writer")
			} else {
				printDebug("closed UnixConn writer")
			}
		} else {
			printDebug("can't close conn writer")
		}
		o.Do(closeChan2)
		close(chan3)
	}()

	<-chan1
	<-chan2
	conn.Close()
	sshConn.Close()

	<-chan3

	printDebug("closed socket connection")

	return nil
}
