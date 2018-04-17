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

	conn.Write([]byte("C{\"func\":\"mul\",\"data\":\"my_data\"}\n")) // always add \n at the end!
	reader := bufio.NewReader(conn)
	message, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	fmt.Println("Answer was recieved:", message)

	conn.Write([]byte("C{\"func\":\"div\",\"data\":\"div_data\"}\n")) // always add \n at the end!
	message, err = reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	fmt.Println("Answer was recieved:", message)

	conn.Close()
}
