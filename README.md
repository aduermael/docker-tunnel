# docker-tunnel

Connect your local Docker client to remote a Docker engine through SSH tunnel.

### Requirements:

- Make sure you can connect to your remote Docker host using public key authentication


### How to install:

```shell
go install github.com/aduermael/docker-tunnel
```

### Usage:

```shell
docker-tunnel [user] host
```

`user` is optional, "root" by defaut.


```shell
🐳  $ docker -v
Docker version 1.12.0, build 8eab29e
```

That's it! Your local Docker client is now connected to your remote Docker engine using an SSH tunnel (only in that particular bash session).

### Exit:

```shell
🐳  $ exit
```
