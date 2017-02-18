# docker-tunnel

Connect your local Docker client to a remote Docker engine using an SSH tunnel.

### Requirements:

- Make sure you can connect to your remote Docker host using SSH public key authentication
- OpenSSH 6.7 minimum required on both sides (use `ssh -V` to check)


### How to install:

```shell
go install github.com/aduermael/docker-tunnel
```

### Usage:

```shell
# `user` is optional, "root" by defaut.
$ docker-tunnel [user] host
```

That's it! Your local Docker client is now connected to your remote Docker engine using an SSH tunnel (only in that particular bash session).

```shell
🐳  $ docker version -f "{{.Server.KernelVersion}}"
4.4.0-42-generic
```

### Exit:

```shell
🐳  $ exit
```
