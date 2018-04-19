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
	reader := bufio.NewReader(conn)

	// login first if you have already registered
	conn.Write([]byte("I{\"login\":\"pavel\",\"pass\":\"lol\"}\n"))
	msgType, err := reader.ReadByte()
	message, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	if msgType == 'E' {
		panic(string(message))
	}
	fmt.Println(string(message))

	// we assume that client can execute no more than one function
	conn.Write([]byte("P{\"func\":\"mul\"}\n")) // always add \n at the end!
	conn.Write([]byte("R\n"))

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		fmt.Println("Task was recieved:", message)
		// Now you can check type of message by switch message[0]
		// and then unmarshal it properly with server's rules
		// after that all data's structure is depended
		// on another client

		conn.Write([]byte("Answer for " + message))
	}
	conn.Close()
}