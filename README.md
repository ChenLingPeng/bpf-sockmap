# bpf-sockmap

bpf-sockmap uses [BPF_PROG_TYPE_SK_SKB](https://lwn.net/Articles/731133/) BPF programs to create a simple telnet echo server. It is heavily inspired by the [Cloudflare blog](https://blog.cloudflare.com/sockmap-tcp-splicing-of-the-future/).

## Quickstart

[Vagrant](https://www.vagrantup.com/) can be used to spin up a virtual environment to test the BPF programs. The environment depends on [VirtualBox](https://www.virtualbox.org/wiki/Downloads) but other [providers](https://www.vagrantup.com/docs/providers/) exist.

This example start two server listen on 0.0.0.0:12345 & 127.0.0.1:12346. 
Client connect to 12345 and server will redirect data from client to 12346 using sockmap/

```
$ vagrant plugin install vagrant-reload
$ vagrant box list | grep ubuntu/bionic64 || vagrant box add ubuntu/bionic64
$ vagrant up
$ vagrant ssh
# we are assuming commands are run from within the vagrant vm from here
$ cd /vagrant
$ make build
$ make run
...
2019/04/03 00:53:12 listening on address: 0.0.0.0:12345
# in another terminal watch debug output
$ sudo cat /sys/kernel/debug/tracing/trace_pipe
# in yet another terminal start a telnet session
$ telnet 127.0.0.1 12345
Trying 127.0.0.1...
Connected to 127.0.0.1.
Escape character is '^]'.
Hello!
answer 1: Hello!
Bye
answer 2: Bye
^]q

telnet> q
Connection closed.
```

## Debug

The generated object file can be inspected using `llvm-objdump`

```
llvm-objdump -S ./bpf/bpf_sockmap.o
```
