package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

// This is a simple TCP server with it's own communication protocol over TCP
// All messages to server should end with '\n' character, server does the same rule

// There are several message types: P, R, C, E, D, U, I, O, M, S
// P - POST  - notifies the server that it can handle the function described in postQuery struct
//             Ex.: "P{"func":"division"}\n"
// R - READY - notifies the server that it is ready to handle it's function
//             Ex.: "R\n"
// C - CALC  - asks server to calculate function with parameters desribed in calcQuery.
//             Server does not actually calculate anything, it's task just
//             to transfer all data to appropriate R conn, get back the answer and pass it to C sender
//             Ex.: "C{\"func\":\"mul\",\"data\":\"my_data\"}\n"
// E - ERROR - error message
//             Ex.: "ESomething bad happened!\n"
// D - DONE  - contains result of CALC message if it was processed successfully
//             Ex.: "D{"Your data"}\n"
// U - UP    - registers (signs up) client on server with parameters in logQuery struct
//             Ex.:
// I - IN    - logins (signs in) client on server with parameters in logQuery struct
//             Ex.:
// O - OK    - always sends from server after client's U or I message if it was succsessful
// M - MSG   - sends private message to another client with parameters in msgQuery struct
//             Ex.:
// S - STR   - streams message to everyone
//             Ex.: "Shi there!\n"

// The server can work with income messages of types P, R, C, U, I, M, S
// The client should be able to handle O, C, D, E, M -messages

type postQuery struct {
	Function string `json:"func"`
}

type calcQuery struct {
	Function string          `json:"func"`
	Data     json.RawMessage `json:"data"`
}

type logQuery struct {
	Login    string `json:"login"`
	Password string `json:"pass"`
}

var (
	registeredUsers = "./database.txt"         // TODO: redis?
	activeUsers     = map[string]*inOutChans{} // login -> ut's conn chans
	funcMap         = map[string]string{}      // func -> login that can handle it
)

type msgQuery struct {
	Receiver string `json:"rec"`
	Message  string `json:"msg"`
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

// Logins or registers user with login and pass
func loginOrRegister(login, pass string, reg bool, chans *inOutChans) bool /*, error*/ {
	file, err := os.Open(registeredUsers)
	defer file.Close()
	if err != nil {
		panic(err)
	}
	fileReader := bufio.NewReader(file)
	for {
		content, err := fileReader.ReadString('\n')
		if err != nil { // os.EOF
			break
		}
		logPass := strings.Split(content, "---")
		if logPass[0] == login { // if this login was taken
			if reg {
				return false // login isn't unique
			}
			if logPass[1] == pass+"\n" {
				activeUsers[login] = chans // add to active users
				return true                // succsessful login
			}
			return false // wrong password

		}
	}
	if reg {
		file.WriteString(login + "---" + pass + "\n") // add to db
		return true                                   // login is unique
	}
	return false // wrong login
}

// Closes conn and deletes func from funcMap if exists
func closeConn(conn net.Conn, login *string) {
	name := conn.RemoteAddr().String()
	for key, value := range funcMap {
		if value == *login {
			delete(funcMap, key)
			break
		}
	}
	delete(activeUsers, *login)
	conn.Close()
	fmt.Println(name, "disconnected")
}

func handleConnection(conn net.Conn, mu, mu2 *sync.Mutex) {
	name := conn.RemoteAddr().String()
	var login string

	fmt.Printf("%+v connected\n", name)

	scanner := bufio.NewReader(conn)
	dataChan := make(chan string)
	chans := &inOutChans{in: &dataChan, out: nil}

	defer closeConn(conn, &login)
LOOP:
	for {
		msgType, err := scanner.ReadByte()
		if err != nil {
			fmt.Println("Error: can't read message type")
			conn.Write([]byte("ECan't read message type!\r\n"))
			break
		}
		buf, err := scanner.ReadBytes('\n') // because of this, please always add '\n' at the end of your message
		if err != nil {
			fmt.Println("Error: can't read message content")
			conn.Write([]byte("ECan't read message content!\r\n"))
			break
		}
		text := filterNewLines(string(buf))
		fmt.Printf("type: %s, data: %s\n", string(msgType), text)

		switch msgType {
		case 'U': // SIGN UP
			fmt.Println(name, "- UP request")
			r := &logQuery{}
			err = json.Unmarshal(buf, r)
			if err != nil {
				fmt.Println("Can't unmarshal buf")
				conn.Write([]byte("EServer can't unmarshal message content!\r\n"))
				break LOOP
			}
			result := loginOrRegister(r.Login, r.Password, true, nil)
			if !result || r.Login == "" {
				fmt.Println("Attempt to register with not unique login")
				conn.Write([]byte("EThis login has already been taken\r\n"))
				break LOOP
			} else {
				fmt.Println(name, "- registered succsessfully")
				conn.Write([]byte("OSuccsessfully signed up\r\n"))
			}
			//login = r.Login
			continue LOOP

		case 'I': // SIGN IN
			fmt.Println(name, "- IN request")
			l := &logQuery{}
			err = json.Unmarshal(buf, l)
			if err != nil {
				fmt.Println("Can't unmarshal buf")
				conn.Write([]byte("EServer can't unmarshal message content!\r\n"))
				break LOOP
			}
			result := loginOrRegister(l.Login, l.Password, false, chans)
			if !result {
				fmt.Println("Attempt to login in with wrong login/pass")
				conn.Write([]byte("EWrong login/pass\r\n"))
				break LOOP
			} else {
				fmt.Println(name, "- logged in succsessfully")
				conn.Write([]byte("OSuccsessfully logged in\r\n"))
			}
			login = l.Login
			continue LOOP

		case 'M': // MESSAGE
			fmt.Println(name, "- MSG request")
			if login == "" { //check login
				fmt.Println("Unlogged CALC request")
				conn.Write([]byte("EYou should login first!\r\n"))
				break LOOP
			}
			// TODO

		case 'S': // STREAM
			fmt.Println(name, "- STR request")
			if login == "" { //check login
				fmt.Println("Unlogged CALC request")
				conn.Write([]byte("EYou should login first!\r\n"))
				break LOOP
			}
			// TODO

		case 'C': // CALC - conn asks to calculate calcQuery.Function with parameters stored in calcQuery.Data
			fmt.Println(name, "- CALC request")
			if login == "" { //check login
				fmt.Println("Unlogged CALC request")
				conn.Write([]byte("EYou should login first!\r\n"))
				break LOOP
			}
			c := &calcQuery{}
			err = json.Unmarshal(buf, c) // get params
			if err != nil {
				fmt.Println("Can't unmarshal buf")
				conn.Write([]byte("EServer can't unmarshal message content!\r\n"))
				break LOOP
			}

			mu.Lock()
			handler, ok := funcMap[c.Function] // find func
			if !ok {
				mu.Unlock()
				fmt.Println("Can't find function", c.Function)
				conn.Write([]byte("EThis function wasn't registered on server!\r\n"))
				continue LOOP
			}
			mu2.Lock() // mutex here, so nobody can't rebind outChan until operation is done
			activeUsers[handler].out = &dataChan
			sendChan := activeUsers[handler].in
			mu.Unlock()

			*sendChan <- string(c.Data) + "\n" // send params to func holder
			answer := <-dataChan               // get answer
			mu2.Unlock()

			fmt.Println("Operation was done")
			conn.Write([]byte(answer)) // send answer to conn
			continue LOOP

		case 'P': // POST - conn tries to declare it's postQuery.Function on server
			fmt.Println(name, "- POST request")
			if login == "" { //check login
				fmt.Println("Unlogged POST request")
				conn.Write([]byte("EYou should login first!\r\n"))
				break LOOP
			}

			u := &postQuery{}
			err = json.Unmarshal(buf, u)
			if err != nil {
				fmt.Println("Can't unmarshal buf")
				conn.Write([]byte("ECan't unmarshal buf!\r\n"))
				continue LOOP
			}

			mu.Lock()
			if _, ok := funcMap[u.Function]; ok { // if func with this name is already in a map
				fmt.Println("Function with name", u.Function, "is already exists in map!")
				conn.Write([]byte("EFunction with this name is already exists!\r\n"))
				mu.Unlock()
				break LOOP
			}
			funcMap[u.Function] = login
			mu.Unlock()
			fmt.Printf("Added to map:\n%v\n", funcMap)

		case 'R': // READY - conn is now ready to execute others CALC requests
			fmt.Println(name, "- READY request")
			if login == "" { //check login
				fmt.Println("Unlogged READY request")
				conn.Write([]byte("EYou should login first!\r\n"))
				break LOOP
			}

			for {
				data := <-dataChan // get values from chan
				conn.Write([]byte("C" + data))
				ans, err := scanner.ReadBytes('\n')
				if err != nil { // Can be caused by closing READY connection
					fmt.Println("Error: can't read answer content")
					conn.Write([]byte("ECan't read answer content!\r\n"))
					*chans.out <- "ECan't read answer content!\r\n"
					break LOOP
				}
				*chans.out <- "D" + string(ans) // send ans to out chan
			}
		default:
			fmt.Println("Error: wrong message type")
			conn.Write([]byte("EWrong message type!\r\n"))
			break LOOP
		}

	}
}

func main() {
	//funcMap := make(map[string]string) // function -> user that can handle it //input and output chans
	muMap := &sync.Mutex{} // locks all r/w operations with funcMap
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
		go handleConnection(conn, muMap, muChans)
	}
}
