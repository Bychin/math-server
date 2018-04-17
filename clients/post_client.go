package main

import (
	"bufio"
	"fmt"
	"net"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		panic(err)
	}

	conn.Write([]byte("P{\"func\":\"mul\"}\n")) // always add \n at the end!
	//conn.Write([]byte("P{\"func\":\"div\"}\n")) // always add \n at the end!
	conn.Write([]byte("R\n"))

	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		fmt.Println("Task was recieved:", message)

		conn.Write([]byte("Answer for " + message))
	}
	conn.Close()
}
