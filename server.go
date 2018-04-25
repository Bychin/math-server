package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
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
//             Ex.: ""

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

type msgQuery struct {
	Receiver string `json:"rec"`
	Message  string `json:"msg"`
}

type inOutChans struct {
	in  *chan string
	out *chan string
}

type connChans struct {
	chans *inOutChans
	conn  net.Conn
}

var (
	registeredUsers = "./database.txt"        // TODO: redis?
	activeUsers     = map[string]*connChans{} // login -> it's conn chans and conn itself
	funcMap         = map[string]string{}     // func -> login that can handle it
)

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
func loginOrRegister(login, pass string, reg bool, chans *connChans) bool /*, error*/ {
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
	log.Println(name, "- [OK]: disconnected")
}

func handleConnection(conn net.Conn, mu, mu2 *sync.Mutex) {
	name := conn.RemoteAddr().String()
	log.Println(name, "- [OK]: connected")

	scanner := bufio.NewReader(conn)
	dataChan := make(chan string)
	chans := &connChans{
		conn:  conn,
		chans: &inOutChans{in: &dataChan, out: nil},
	}
	var login string
	defer closeConn(conn, &login)
LOOP:
	for {
		msgType, err := scanner.ReadByte()
		if err == io.EOF {
			log.Println(name, "- [EOF]")
			break
		} else if err != nil {
			log.Println(name, "- [ERR]: can't read message type")
			conn.Write([]byte("ECan't read message type!\r\n"))
			break
		}
		buf, err := scanner.ReadBytes('\n') // because of this, please always add '\n' at the end of your message
		if err != nil {
			log.Println(name, "- [ERR]: can't read message content")
			conn.Write([]byte("ECan't read message content!\r\n"))
			break
		}
		text := filterNewLines(string(buf))
		log.Printf("%s - type: %s, data: %s\n", name, string(msgType), text)

		switch msgType {
		case 'U': // SIGN UP
			log.Println(name, "- [OK]: UP request")
			r := &logQuery{}
			err = json.Unmarshal(buf, r)
			if err != nil {
				log.Println(name, "- [ERR]: can't unmarshal buf")
				conn.Write([]byte("EServer can't unmarshal message content!\r\n"))
				break LOOP
			}
			result := loginOrRegister(r.Login, r.Password, true, nil)
			if !result || r.Login == "" {
				log.Println(name, "- [ERR]: attempt to register with not unique login")
				conn.Write([]byte("EThis login has already been taken\r\n"))
				break LOOP
			} else {
				log.Println(name, "- [OK]: registered succsessfully")
				conn.Write([]byte("OSuccsessfully signed up\r\n"))
			}
			// login = r.Login
			continue LOOP

		case 'I': // SIGN IN
			log.Println(name, "- [OK]: IN request")
			l := &logQuery{}
			err = json.Unmarshal(buf, l)
			if err != nil {
				log.Println(name, "- [ERR]: can't unmarshal buf")
				conn.Write([]byte("EServer can't unmarshal message content!\r\n"))
				break LOOP
			}
			result := loginOrRegister(l.Login, l.Password, false, chans)
			if !result {
				log.Println(name, "- [ERR]: attempt to login in with wrong login/pass")
				conn.Write([]byte("EWrong login/pass\r\n"))
				break LOOP
			} else {
				log.Println(name, "- [OK]: logged in succsessfully")
				conn.Write([]byte("OSuccsessfully logged in\r\n"))
			}
			login = l.Login
			continue LOOP

		case 'M': // MESSAGE
			log.Println(name, "- [OK]: MSG request")
			if login == "" { //check login
				log.Println(name, "- [ERR]: unlogged MSG request")
				conn.Write([]byte("EYou should login first!\r\n"))
				break LOOP
			}
			m := &msgQuery{}
			err = json.Unmarshal(buf, m)
			if err != nil {
				log.Println(name, "- [ERR]: can't unmarshal buf")
				conn.Write([]byte("EServer can't unmarshal message content!\r\n"))
				break LOOP
			}
			mu2.Lock()
			resChan, ok := activeUsers[m.Receiver]
			if !ok { // if user-receiver is not logged in
				log.Println(name, "- [ERR]: user", m.Receiver, "is not logged in!")
				conn.Write([]byte("EReceiver is not logged in!\r\n"))
				mu2.Unlock()
				continue LOOP
			}
			m.Receiver = login
			msg, _ := json.Marshal(m)
			resChan.conn.Write([]byte("M"))
			resChan.conn.Write(msg) // error check
			resChan.conn.Write([]byte("\n"))
			mu2.Unlock()
			continue LOOP

		case 'S': // STREAM
			log.Println(name, "- [OK]: STR request")
			if login == "" { //check login
				log.Println(name, "- [ERR]: unlogged STR request")
				conn.Write([]byte("EYou should login first!\r\n"))
				break LOOP
			}

			mu2.Lock()
			for key, value := range activeUsers {
				if key == login {
					continue
				}
				value.conn.Write([]byte("M"))
				value.conn.Write(buf) // error check here?
				//value.conn.Write([]byte("\n"))
			}
			mu2.Unlock()
			continue LOOP

		case 'C': // CALC - conn asks to calculate calcQuery.Function with parameters stored in calcQuery.Data
			log.Println(name, "- [OK]: CALC request")
			if login == "" { //check login
				log.Println(name, "- [ERR]: unlogged CALC request")
				conn.Write([]byte("EYou should login first!\r\n"))
				break LOOP
			}
			c := &calcQuery{}
			err = json.Unmarshal(buf, c) // get params
			if err != nil {
				log.Println(name, "- [ERR]: can't unmarshal buf")
				conn.Write([]byte("EServer can't unmarshal message content!\r\n"))
				break LOOP
			}

			mu.Lock()
			handler, ok := funcMap[c.Function] // find func
			if !ok {
				mu.Unlock()
				log.Println(name, "- [ERR]: can't find function", c.Function)
				conn.Write([]byte("EThis function wasn't registered on server!\r\n"))
				continue LOOP
			}
			mu2.Lock() // mutex here, so nobody can't rebind outChan until operation is done
			activeUsers[handler].chans.out = &dataChan
			sendChan := activeUsers[handler].chans.in
			mu.Unlock()

			*sendChan <- string(c.Data) + "\n" // send params to func holder
			answer := <-dataChan               // get answer
			mu2.Unlock()

			log.Println(name, "- [OK]: operation was done")
			conn.Write([]byte(answer)) // send answer to conn
			continue LOOP

		case 'P': // POST - conn tries to declare it's postQuery.Function on server
			log.Println(name, "- [OK]: POST request")
			if login == "" { //check login
				log.Println(name, "- [ERR]: unlogged POST request")
				conn.Write([]byte("EYou should login first!\r\n"))
				break LOOP
			}

			u := &postQuery{}
			err = json.Unmarshal(buf, u)
			if err != nil {
				log.Println(name, "- [ERR]: can't unmarshal buf")
				conn.Write([]byte("ECan't unmarshal buf!\r\n"))
				continue LOOP
			}

			mu.Lock()
			if _, ok := funcMap[u.Function]; ok { // if func with this name is already in a map
				log.Println(name, "- [ERR]: function with name", u.Function, "is already exists in map!")
				conn.Write([]byte("EFunction with this name is already exists!\r\n"))
				mu.Unlock()
				break LOOP
			}
			funcMap[u.Function] = login
			mu.Unlock()
			log.Printf("%s - [OK]: added to map:\n%v\n", name, funcMap)

		case 'R': // READY - conn is now ready to execute others CALC requests
			log.Println(name, "- [OK]: READY request")
			if login == "" { //check login
				log.Println(name, "- [ERR]: unlogged READY request")
				conn.Write([]byte("EYou should login first!\r\n"))
				break LOOP
			}

			for {
				data := <-dataChan // get values from chan
				conn.Write([]byte("C" + data))
				ans, err := scanner.ReadBytes('\n')
				if err != nil { // Can be caused by closing READY connection
					log.Println(name, "- [ERR]: can't read answer content")
					conn.Write([]byte("ECan't read answer content!\r\n"))
					*chans.chans.out <- "ECan't read answer content!\r\n"
					break LOOP
				}
				*chans.chans.out <- "D" + string(ans) // send ans to out chan
			}
		default:
			log.Println(name, "- [ERR]: wrong message type")
			conn.Write([]byte("EWrong message type!\r\n"))
			break LOOP
		}
	}
}

func main() {
	logFile, err := os.OpenFile("./logs/"+time.Now().String()+".log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("server - [ERR]: Cannot open log file: %v", err)
	}
	defer logFile.Close()
	//mw := io.MultiWriter(os.Stdout, logFile)
	//log.SetOutput(mw)
	log.Println("Starting server...")

	muMap := &sync.Mutex{}   // locks all r/w operations with funcMap
	muChans := &sync.Mutex{} // locks all r/w operations with activeUsers
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
