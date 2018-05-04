package main

import (
	"bufio"
	"net"
	"sync"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		panic(err)
	}
	reader := bufio.NewReader(conn)

	// login first if you have already registered
	Login("pavel3", conn, *reader)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go StreamMessages("pavel3", conn)
	go ReadMessages(*reader)

	wg.Wait()
	conn.Close()
}
