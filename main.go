package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

			userAtHost := args[0]
			// root user by default
			if !strings.Contains(userAtHost, "@") {
				userAtHost = "root@" + userAtHost
			}

			tunnelProcess, socketPath := createTunnel(userAtHost)
			defer tunnelProcess.Kill()
			defer os.RemoveAll(socketPath)

			if proxyMode {
				fmt.Println("listening on port 2375")
				err := http.ListenAndServe(":2375", proxy(socketPath))
				if err != nil {
					log.Fatalln(err)
				}
				return
			}

			// proxyMode == false
			// open shell session, connected to remote Docker host

			os.Setenv("PS1", "🐳  $ ")
			os.Setenv("DOCKER_HOST", "unix://"+socketPath)

			sh := exec.Command(shell)
			sh.Stdout = os.Stdout
			sh.Stderr = os.Stderr
			sh.Stdin = os.Stdin

			err := sh.Run()
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

func createTunnel(userAtHost string) (process *os.Process, socketPath string) {
	socketPath = tmpSocketPath()

	args := []string{
		"-nNT",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "StrictHostKeyChecking=no",
		"-L", socketPath + ":/var/run/docker.sock",
	}

	// check if custom ssh id path should be used
	if sshIdentityFile != "" {
		args = append(args, "-i", sshIdentityFile)
	}

	args = append(args, userAtHost)

	cmd := exec.Command("ssh", args...)
	err := cmd.Start()
	if err != nil {
		log.Fatalln(err.Error())
	}

	process = cmd.Process
	return
}

func tmpSocketPath() string {
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	return filepath.Join(os.TempDir(), "docker-"+hex.EncodeToString(randBytes)+".sock")
}

func proxy(socketPath string) *httputil.ReverseProxy {
	u, err := url.Parse("http://unix.sock")
	if err != nil {
		log.Fatalln(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(u)

	tr := &http.Transport{
		Dial: func(proto, addr string) (conn net.Conn, err error) {
			return net.Dial("unix", socketPath)
		},
	}
	proxy.Transport = tr

	return proxy
}
