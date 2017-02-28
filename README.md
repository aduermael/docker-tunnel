# docker-tunnel

Connect to a remote Docker host using an SSH tunnel.

### Requirements:

- Make sure you can connect to your remote Docker host using SSH public key authentication
- OpenSSH 6.7 minimum required at least on the server side.

### How to install:

- **docker-tunnel** can be installed directly on your host like this ([Go](https://golang.org/doc/install) has to be installed):

	```bash
	$ go install github.com/aduermael/docker-tunnel
	```
- You can also get the Docker image:
	
	```bash
	$ docker pull aduermael/docker-tunnel
	```

### Usage:

**docker-tunnel** has a small command line interface:

```bash
# type this if you installed directly on your host:
$ docker-tunnel
# or this if using the Docker image:
$ docker run --rm aduermael/docker-tunnel

# in both cases, you'll see something like this:
Usage:
  docker-tunnel [user@]host [flags]

Flags:
  -p, --proxy          proxy mode (don't start shell session)
  -s, --shell string   shell to open session (default "bash")
  -i, --sshid string   path to private key

```

**docker-tunnel** can be used in 2 different modes:

- **shell mode** (default): opens a shell session, bash by default but a different one can be requested using `-s` flag. From within this shell, all Docker commands are sent to the remote Docker host through an established SSH tunnel.

- **proxy mode** (using `-p` flag): exposes a Docker remote API on port 2375, proxying all requests over SSH to the remote Docker host.

In both modes, the `-i` flag can be used to give the location of your ssh identity file (private key). 

### Examples

Run container acting as a Docker remote API proxy to reach remote Docker host.

```bash
$ docker run --rm -v ~/.ssh/id_rsa:/ssh_id -p 127.0.0.1:2375:2375 \
aduermael/docker-tunnel 138.88.888.888 -i /ssh_id -p

# now in a different shell session you can do:
export DOCKER_HOST=tcp://127.0.0.1:2375
# all docker commands will now target the remote side (through the proxy)
```

Open a bash session to run containers on a remote Docker host, using files from your local environment:

```bash
$ docker-tunnel user@138.88.888.888
```