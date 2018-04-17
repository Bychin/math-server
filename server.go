package main

import (
	"bufio"
	"encoding/json"
	//"errors"
	"fmt"
	"net"
	//"reflect"
	"strings"
	"sync"
	"time"
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

// Closes conn and deletes func from funcMap if exists
func closeConn(conn net.Conn, funcMap map[string]*inOutChans, chans map[string]*inOutChans) {
	name := conn.RemoteAddr().String()
	for key := range chans {
		if _, ok := funcMap[key]; ok {
			delete(funcMap, key)
		}
	}

	conn.Close()
	fmt.Println(name, "disconnected")
}

/*
for { // how to close? (with func remove from map)
				data := <-dataChan // get values from chan

				conn.Write([]byte("C" + data))
				ans, err := scanner.ReadBytes('\n')
				if err != nil { // Can be caused by closing READY connection
					fmt.Println("Error: can't read answer content")
					conn.Write([]byte("ECan't read answer content!\n\r"))
					*chans.out <- "ECan't read answer content!\n\r"
					break LOOP
				}
				*chans.out <- "D" + string(ans) // send ans to out chan
			}
*/

// listens chans.in which is blong to conn
func listenConn(pair *inOutChans, conn net.Conn, connReader *bufio.Reader) {
LOOP:
	for {
		select {
		case data := <-*pair.in:
			conn.Write([]byte("C" + data)) // calcQuery format
			ans, err := connReader.ReadBytes('\n')
			if err != nil { // Can be caused by closing READY connection
				fmt.Println("Error: can't read answer content")
				conn.Write([]byte("ECan't read answer content!\n\r"))
				*pair.out <- "ECan't read answer content!\n\r"
				break LOOP
			}
			*pair.out <- string(ans) // send ans to out chan
		case <-time.After(time.Millisecond):
		}
	}
}

func handleConnection(conn net.Conn, funcMap map[string]*inOutChans, mu, mu2 *sync.Mutex) {
	name := conn.RemoteAddr().String()

	fmt.Printf("%+v connected\n", name)

	scanner := bufio.NewReader(conn)
	//dataChan := make(chan string)
	chans := make(map[string]*inOutChans) //&inOutChans{in: &dataChan, out: nil}
	defer closeConn(conn, funcMap, chans) //!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
LOOP:
	for {
		msgType, err := scanner.ReadByte()
		if err != nil {
			fmt.Println("Error: can't read message type")
			conn.Write([]byte("ECan't read message type!\n\r"))
			break
		}
		buf, err := scanner.ReadBytes('\n') // because of this, please always add '\n' at the end of your message
		if err != nil {
			fmt.Println("Error: can't read message content")
			conn.Write([]byte("ECan't read message content!\n\r"))
			break
		}
		text := filterNewLines(string(buf))
		fmt.Printf("type: %s, data: %s\n", string(msgType), text)

		switch msgType {
		case 'C': // CALC - conn asks to calculate calcQuery.Function with parameters stored in calcQuery.Data
			fmt.Println(name, "- CALC request")

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
			dataChan := make(chan string) // create new out chan for getiing the result
			mu2.Lock()                    // mutex here, so nobody can rebind inOutChans.out
			funcMap[c.Function].out = &dataChan
			sendChan := funcMap[c.Function].in
			mu.Unlock()

			//*sendChan <- string(c.Function) + "\n" // send func name to func holder (so conn holder can chose which func to execute)
			*sendChan <- string(buf) + "\n" // send params to func holder
			answer := <-dataChan            // get answer
			mu2.Unlock()

			fmt.Println("Operation was done, closing conn")
			conn.Write([]byte(answer)) // send answer to conn
			break LOOP

		case 'P': // POST - conn tries to declare it's postQuery.Function on server
			fmt.Println(name, "- POST request")

			u := &postQuery{}
			err = json.Unmarshal(buf, u)
			if err != nil {
				fmt.Println("Can't unmarshal buf")
				conn.Write([]byte("ECan't unmarshal buf!\n\r"))
				continue LOOP
			}

			mu.Lock()
			if _, ok := funcMap[u.Function]; ok { // if func with this name is already in a map
				fmt.Println("Function with name", u.Function, "is already exists in map!")
				conn.Write([]byte("EFunction with this name is already exists!\n\r"))
				mu.Unlock()
				break LOOP
			}
			dataChan := make(chan string) // create new in-chan for handling new function
			chans[u.Function] = &inOutChans{in: &dataChan, out: nil}
			funcMap[u.Function] = chans[u.Function]
			mu.Unlock()
			fmt.Printf("Added to map:\n%v\n", funcMap)

		case 'R': // READY - conn is now ready to execute others CALC requests
			fmt.Println(name, "- READY request")

			// for pair in chans listen pair.in
			for _, value := range chans {
				go listenConn(value, conn, scanner)
				// how to close conn if break?
			}

			/*for { // how to close? (with func remove from map)
				data := <-dataChan // get values from chan

				conn.Write([]byte("C" + data))
				ans, err := scanner.ReadBytes('\n')
				if err != nil { // Can be caused by closing READY connection
					fmt.Println("Error: can't read answer content")
					conn.Write([]byte("ECan't read answer content!\n\r"))
					*chans.out <- "ECan't read answer content!\n\r"
					break LOOP
				}
				*chans.out <- "D" + string(ans) // send ans to out chan
			}*/
		default:
			fmt.Println("Error: wrong message type")
			conn.Write([]byte("EWrong message type!\n\r"))
			break LOOP
		}

	}
}

func main() {
	funcMap := make(map[string]*inOutChans) // function -> input and output chans
	muMap := &sync.Mutex{}                  // locks all r/w operations with funcMap
	muChans := &sync.Mutex{}
	listner, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	for {
		conn, err := listner.Accept()
		if err != nil {
			panic(err)
		}
		go handleConnection(conn, funcMap, muMap, muChans)
	}
}
