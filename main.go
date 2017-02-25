package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	tunnelProcess, socketPath := createTunnel()
	defer tunnelProcess.Kill()
	defer os.RemoveAll(socketPath)

	os.Setenv("PS1", "🐳  $ ")
	os.Setenv("DOCKER_HOST", "unix://"+socketPath)

	bash := exec.Command("bash")
	bash.Stdout = os.Stdout
	bash.Stderr = os.Stderr
	bash.Stdin = os.Stdin

	bash.Run()
}

func createTunnel() (process *os.Process, socketPath string) {
	socketPath = tmpSocketPath()

	user := ""
	host := ""
	if len(os.Args) == 2 {
		host = os.Args[1]
		user = "root"
	} else if len(os.Args) == 3 {
		user = os.Args[1]
		host = os.Args[2]
	} else {
		log.Fatalln("usage: docker-tunnel [user] host")
	}

	cmd := exec.Command("ssh", "-nNT",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "StrictHostKeyChecking=no",
		"-L", socketPath+":/var/run/docker.sock", user+"@"+host)

	err := cmd.Start()
	if err != nil {
		log.Fatalln("ERROR:", err.Error())
	}

	process = cmd.Process
	return
}

func tmpSocketPath() string {
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	return filepath.Join(os.TempDir(), "docker-"+hex.EncodeToString(randBytes)+".sock")
}
