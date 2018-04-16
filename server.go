package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
)

type postQuery struct {
	Function string `json:"func"`
}

type calcQuery struct {
	Function string          `json:"func"`
	Data     json.RawMessage `json:"data"`
}

type inOutChans struct {
	in  *chan string
	out *chan string
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

func handleConnection(conn net.Conn, funcMap map[string]*inOutChans, mu *sync.Mutex) {
	name := conn.RemoteAddr().String()

	fmt.Printf("%+v connected\n", name)
	conn.Write([]byte("Hello, " + name + ", type \"Exit\" to exit.\n"))
	defer conn.Close()

	scanner := bufio.NewReader(conn)

	dataChan := make(chan string)
	chans := &inOutChans{in: &dataChan, out: nil}
LOOP:
	for {
		msgType, err := scanner.ReadByte()
		if err != nil {
			fmt.Println("Error: can't read message type")
			conn.Write([]byte("ECan't read message type!\n\r"))
			fmt.Println(name, "disconnected")
			break
		}
		buf, err := scanner.ReadBytes('\n')
		if err != nil {
			fmt.Println("Error: can't read message content")
			conn.Write([]byte("ECan't read message content!\n\r"))
			fmt.Println(name, "disconnected")
			break
		}
		text := filterNewLines(string(buf))
		fmt.Printf("type: %s, data: %s\n", string(msgType), text)

		switch msgType {
		case 'C':
			fmt.Println("Calc type")
			c := &calcQuery{}
			err = json.Unmarshal(buf, c) // get params
			if err != nil {
				fmt.Println("Can't unmarshal buf")
				conn.Write([]byte("EServer can't unmarshal message content!\n\r"))
				break LOOP
			}

			mu.Lock()
			_, ok := funcMap[c.Function] // find func
			if !ok {
				mu.Unlock()
				fmt.Println("Can't find function", c.Function)
				conn.Write([]byte("EYour function wasn't registered on server!\n\r"))
				break LOOP
			}
			funcMap[c.Function].out = &dataChan
			sendChan := funcMap[c.Function].in
			mu.Unlock()

			// mutex here, so nobody can rebind outChan
			*sendChan <- string(c.Data) // send params to func holder
			answer := <-dataChan        // get answer

			// send answer to conn
			fmt.Println("Operation was done, closing conn")
			conn.Write([]byte("D" + answer))
			break LOOP
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
			funcMap[u.Function] = chans //&inOutChans{in: dataChan, out: nil} //conn.RemoteAddr()
			mu.Unlock()
			fmt.Printf("Added to map:\n%v\n", funcMap)

			if text != "" {
				fmt.Println(name, "enters", text)
				conn.Write([]byte("You enter " + text + "\n\r"))
			}
		case 'R':
			for { // how to close? (with func remove from map)
				fmt.Println("conn is ready to go!")

				data := <-dataChan // get values from chan
				conn.Write([]byte("C" + data))
				ans, err := scanner.ReadBytes('\n')
				if err != nil {
					fmt.Println("Error: can't read answer content")
					conn.Write([]byte("ECan't read answer content!\n\r"))
					fmt.Println(name, "disconnected")
					//break LOOP
				}
				*chans.out <- string(ans) // send ans to out chan
			}
		default:
			fmt.Println("Error: wrong message type")
			conn.Write([]byte("EWrong message type!\n\r"))
			break LOOP
		}

	}
}

func main() {
	funcMap := make(map[string]*inOutChans) // function -> remote addres
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
