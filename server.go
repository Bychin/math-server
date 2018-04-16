package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	///"strings"
)

type postQuery struct {
	Function string `json:"func"`
}

type calcQuery struct {
	Function string `json:"func"`
	// data ???
}

// Filters s to get rid of all escape characters as '\n', '\r', etc...
func filterNewLines(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case 0x000A, 0x000B, 0x000C, 0x000D, 0x0085, 0x2028, 0x2029:
			return -1
		default:
			return r
		}
	}, s)
}

func handleConnection(conn net.Conn, funcMap map[string]net.Addr, mu *sync.Mutex) {
	name := conn.RemoteAddr().String()

	fmt.Printf("%+v connected\n", name)
	conn.Write([]byte("Hello, " + name + ", type \"Exit\" to exit.\n"))

	defer conn.Close()

	//	var msgType byte
	scanner := bufio.NewReader(conn)
LOOP:
	for {
		msgType, err := scanner.ReadByte()
		if err != nil {
			fmt.Println("Error: can't read meassge type")
			// conn.Write
			break
		}
		buf, err := scanner.ReadBytes('\n')
		if err != nil {
			fmt.Println("Error: can't read message content")
			break
		}
		text := filterNewLines(string(buf))
		fmt.Printf("type: %v, data: %s\n", msgType, text)

		switch msgType {
		case 'C':
			fmt.Println("Calc type")
			// find func
			// get params
			// send params to func holder
			// get answer
			// send them back
		case 'P':
			if text == "" {
				continue
			} else if text == "Close" {
				conn.Write([]byte("Bye\n\r"))
				fmt.Println(name, "disconnected")
				break LOOP
			}

			u := &postQuery{} // change this to escape empty strings
			err = json.Unmarshal(buf, u)
			if err != nil {
				fmt.Println("Can't unmarshal buf")
				// conn.Write error!
				continue LOOP
			}

			mu.Lock()
			funcMap[u.Function] = conn.RemoteAddr()
			mu.Unlock()
			fmt.Printf("Added to map:\n%v\n", funcMap)

			if text != "" {
				fmt.Println(name, "enters", text)
				conn.Write([]byte("You enter " + text + "\n\r"))
			}
		default:
			fmt.Println("Error: wrong message type")
			// conn.Write
			break LOOP
		}

	}
}

func main() {
	funcMap := make(map[string]net.Addr) // function -> remote addres
	mu := &sync.Mutex{}
	listner, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	for {
		conn, err := listner.Accept()
		if err != nil {
			panic(err)
		}
		go handleConnection(conn, funcMap, mu)
	}
}
