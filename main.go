package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// path to private key or empty to use default
	sshIdentityFile = ""
	shell           = "bash"
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
