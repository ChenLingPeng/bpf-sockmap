package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"syscall"

	"github.com/dippynark/bpf-sockmap/pkg/sockmap"
)

const (
	defaultPort = 12345
)

func main() {
	// setup sockmap module
	sockmap, err := sockmap.New()
	if err != nil {
		log.Fatalf("failed to create new sockmap module: %s", err)
	}
	log.Print("created new sockmap module")
	defer func() {
		err := sockmap.Close()
		if err != nil {
			log.Fatalf("failed to close sockmap module: %s", err)
		}
	}()

	// listen
	listenAddress := fmt.Sprintf("0.0.0.0:%d", defaultPort)
	l, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatalf("failed to listen on port %d: %s", defaultPort, err)
	}
	defer func() {
		err := l.Close()
		if err != nil {
			log.Fatalf("failed to close socket: %s", err)
		}
	}()
	log.Printf("listening on address: %s", listenAddress)

	go createTcpServer()

	// add accepted connections to sockmap
	for {
		// accept
		conn, err := l.Accept()
		if err != nil {
			log.Fatalf("error accepting: %s", err)
		}

		// retrieve copy of connection file descriptor
		tcpConn, ok := conn.(*net.TCPConn)
		if !ok {
			log.Fatalf("failed to cast connection to TCP connection")
		}
		f, err := tcpConn.File()
		if err != nil {
			log.Fatalf("failed to retrieve copy of the underlying TCP connection file")
		}
		d := f.Fd()

		// update element
		err = sockmap.UpdateSockDescWithIndex(d, 0)
		if err != nil {
			log.Fatalf("failed to update socket descriptor: %s", err)
		}
		go func() {
			nc, err := newConn(sockmap, defaultPort+1)
			if err != nil {
				conn.Close()
				f.Close()
			} else {
				waitForCloseByClient(conn, f, nc)
			}
		}()
	}
}

func newConn(sockmap *sockmap.Sockmap, port int) (net.Conn, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		log.Printf("failed dial remote\n")
		return nil, err
	}
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		log.Fatalf("failed to cast connection to TCP connection")
	}
	fmt.Printf("new conn %s\n", tcpConn.LocalAddr())
	f, err := tcpConn.File()
	if err != nil {
		log.Fatalf("failed to retrieve copy of the underlying TCP connection file")
	}
	d := f.Fd()

	// update element
	err = sockmap.UpdateSockDescWithIndex(d, 1)
	if err != nil {
		log.Fatalf("failed to update socket2 descriptor: %s", err)
	}

	// we don't need two copies of the connection file descriptor so close the copy
	// https://stackoverflow.com/questions/28967701/golang-tcp-socket-cant-close-after-get-file#answer-28968431
	err = syscall.SetNonblock(int(d), true)
	if err != nil {
		log.Fatalf("failed to put file descriptor in non-blocking mode: %s", err)
	}
	err = f.Close()
	if err != nil {
		log.Fatalf("failed to close file descriptor copy: %s", err)
	}
	return conn, nil
}

func createTcpServer() {
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", defaultPort+1))
	if err != nil {
		log.Fatalf("failed to listen on port %d: %s", defaultPort+1, err)
	}
	log.Printf("listening on address: 127.0.0.1:%d\n", defaultPort+1)
	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatalf("error accepting: %s", err)
		}
		log.Printf("handle conn to server %d\n", defaultPort+1)
		go handleConn(c)
	}
}

func handleConn(c net.Conn) {
	defer func() {
		log.Printf("conn from %s closed\n", c.RemoteAddr().String())
		c.Close()
	}()
	round := 0
	for {
		bs := make([]byte, 1024)
		n, err := c.Read(bs)
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Printf("read failed %v\n", err)
		} else {
			round++
			answer := []byte(fmt.Sprintf("answer %d: ", round))
			answer = append(answer, bs[:n]...)
			if _, err := c.Write(answer); err != nil {
				log.Printf("failed write %v\n", err)
			}
		}
	}
}

func waitForCloseByClient(conn net.Conn, f *os.File, nconn net.Conn) {
	fmt.Println("Accepted connection from", conn.RemoteAddr())

	defer func() {
		fmt.Println("Closing connection from", conn.RemoteAddr())
		conn.Close()
		f.Close()
		nconn.Close()
	}()

	buf := make([]byte, 1024)
	for {
		_, err := conn.Read(buf)
		if err == io.EOF {
			fmt.Println("Read error", err.Error())
			return
		}
	}
}
