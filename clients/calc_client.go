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
	conn.Write([]byte("I{\"login\":\"pavel2\",\"pass\":\"lol\"}\n"))
	msgType, err := reader.ReadByte()
	message, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	if msgType == 'E' {
		panic(string(message))
	}
	fmt.Println(string(message))

	conn.Write([]byte("C{\"func\":\"mul\",\"data\":\"my_data\"}\n")) // always add \n at the end!

	answer, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	fmt.Println("Answer was recieved:", answer)
	// Now you can check type of answer by switch message[0]
	// and then unmarshal it properly with server's rules
	// after that all data's structure is depended
	// on another client

	conn.Write([]byte("C{\"func\":\"div\",\"data\":\"div_data\"}\n")) // always add \n at the end!
	answer, err = reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	fmt.Println("Answer was recieved:", answer)
	// Same here

	conn.Close()
}
