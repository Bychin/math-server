package main

import (
	"bufio"
	"encoding/json"
	"errors"
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
// The server can work with income messages of types P, R, C, U, I, M, S
// The client should be able to handle O, C, D, E, M -messages

// P - POST  - notifies the server that it can handle the function described in postQuery struct
//             Ex.: 'P{"func":"division"}\n'
// R - READY - notifies the server that it is ready to handle it's function
//             Ex.: "R\n"
// C - CALC  - asks server to calculate function with parameters desribed in calcQuery.
//             Server does not actually calculate anything, it's task just to transfer all data
//             to appropriate R conn, get back the answer and pass it to C sender
//             Ex.: 'C{"func":"mul","data":"my_data"}\n'
// E - ERROR - error message
//             Ex.: "ESomething bad happened!\n"
// D - DONE  - contains result of CALC message if it was processed successfully
//             Ex.: "D{"Your data"}\n"
// U - UP    - registers (signs up) client on server with parameters in logQuery struct
//             Ex.: 'U{"login":"Pavel","pass":"secret"}\n' - server will check whether login
//             "Pavel" exists and if it's not - add it to registeredUsers with password
// I - IN    - logins (signs in) client on server with parameters in logQuery struct
//             Ex.: 'I{"login":"Pavel","pass":"secret"}\n' - server will check whether login
//             "Pavel" exists in registeredUsers and then will check password
// O - OK    - always sends from server after client's U, I, M, S messages if it was succsessful
// M - MSG   - sends private message to another client with parameters in msgQuery struct
//             Ex.: 'M{"rec":"Pavel","msg":"Hi!"}\n' - server will send message "Hi!" to user
//             with login "Pavel" from activeUsers map
// S - STR   - streams message to everyone, uses msgQuery struct for it, but in Receiver field
//             holds sender's login
//             Ex.: 'S{"rec":"sender","msg":"Hi!"}\n' - server will send message "Hi!" to every user
//             in activeUsers map except sender with M type

const (
	logPath = "./logs/"
)

type postQuery struct {
	Function string `json:"func"`
}

type calcQuery struct {
	Function string `json:"func"`
	Data     []byte `json:"data"`
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

type user struct {
	chans *inOutChans
	conn  net.Conn
}

type Server struct {
	registeredUsers string            // TODO: redis
	activeUsers     map[string]*user  // login -> it's conn chans and conn itself
	funcMap         map[string]string // func -> login that can handle it

	muMap   *sync.Mutex // locks all r/w operations with funcMap // TODO: RWMutex?
	muChans *sync.Mutex // locks all r/w operations with activeUsers
}

func NewServer() *Server {
	return &Server{
		registeredUsers: "./database.txt",
		activeUsers:     make(map[string]*user),
		funcMap:         make(map[string]string),
		muMap:           &sync.Mutex{},
		muChans:         &sync.Mutex{},
	}
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
func (s *Server) loginOrRegister(login, pass string, reg bool, u *user) bool /*, error*/ {
	file, err := os.OpenFile(s.registeredUsers, os.O_RDWR, 0644)
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
				s.muChans.Lock()
				s.activeUsers[login] = u // add to active users
				s.muChans.Unlock()
				return true // succsessful login
			}
			return false // wrong password
		}
	}
	if reg {
		_, err := file.WriteString(login + "---" + pass + "\n") // add to db
		if err != nil {
			panic(err)
		}
		return true // login is unique
	}
	return false // wrong login
}

// Closes conn and deletes func from funcMap if exists
func (s *Server) closeConn(conn net.Conn, login *string) {
	name := conn.RemoteAddr().String()
	s.muMap.Lock()
	for key, value := range s.funcMap {
		if value == *login {
			delete(s.funcMap, key)
			break
		}
	}
	s.muMap.Unlock()
	s.muChans.Lock()
	delete(s.activeUsers, *login)
	s.muChans.Unlock()
	conn.Close()
	log.Println(name, "- [OK]: disconnected")
}

func (s *Server) readMsg(name string, scanner bufio.Reader, conn net.Conn) (byte, []byte, error) {
	msgType, err := scanner.ReadByte()
	if err == io.EOF {
		log.Println(name, "- [EOF]")
		return ' ', nil, err
	} else if err != nil {
		s.logError(conn, name, "can't read message type")
		return ' ', nil, err
	}
	buf, err := scanner.ReadBytes('\n') // because of this, please always add '\n' at the end of your message
	if err != nil {
		s.logError(conn, name, "can't read message content!\r\n")
		return ' ', nil, err
	}
	return msgType, buf, nil
}

func (s *Server) handlePost(conn net.Conn, login string, buf []byte) error {
	name := conn.RemoteAddr().String()

	u := &postQuery{}
	err := json.Unmarshal(buf, u)
	if err != nil {
		s.logError(conn, name, "can't unmarshal buf!\r\n")
		return errors.New("can't unmarshal buf") //continue LOOP
	}

	s.muMap.Lock()
	defer s.muMap.Unlock()
	if _, ok := s.funcMap[u.Function]; ok { // if func with this name is already in a map
		s.logError(conn, name, "function with this name is already exists!\r\n")
		return errors.New("function is already exists in map")
	}
	s.funcMap[u.Function] = login
	log.Printf("%s - [OK]: added to map:\n%v\n\t", name, s.funcMap)
	conn.Write([]byte("OFunction was registered\r\n"))
	return nil
}

func (s *Server) logError(conn net.Conn, name, err string) {
	log.Println(name, "- [ERR]: "+err)
	conn.SetWriteDeadline(time.Now().Add(time.Second))
	conn.Write([]byte("E" + err + "\r\n"))
}

func (s *Server) checkLogin(login, name, msgType string, conn net.Conn) bool {
	if msgType != "" {
		log.Println(name, "- [OK]: "+msgType+" request")
	}
	if login == "" {
		s.logError(conn, name, "you should login first!\r\n")
		return false
	}
	return true
}

func (s *Server) signUpUser(name string, buf []byte, conn net.Conn) error {
	r := &logQuery{}
	err := json.Unmarshal(buf, r)
	if err != nil {
		s.logError(conn, name, "server can't unmarshal message content!\r\n")
		return err
	}
	result := s.loginOrRegister(r.Login, r.Password, true, nil)
	if !result || r.Login == "" {
		s.logError(conn, name, "this login has already been taken\r\n")
		return err
	}

	log.Println(name, "- [OK]: registered succsessfully")
	conn.Write([]byte("OSuccsessfully signed up\r\n"))
	return nil
}

func (s *Server) signInUser(name string, buf []byte, u *user, conn net.Conn) (string, error) {
	l := &logQuery{}
	err := json.Unmarshal(buf, l)
	if err != nil {
		s.logError(conn, name, "server can't unmarshal message content!\r\n")
		return "", err
	}
	result := s.loginOrRegister(l.Login, l.Password, false, u)
	if !result {
		s.logError(conn, name, "wrong login/pass\r\n")
		return "", err
	}

	log.Println(name, "- [OK]: logged in succsessfully")
	conn.Write([]byte("OSuccsessfully logged in\r\n"))
	return l.Login, nil
}

func (s *Server) sendMessage(name, login string, buf []byte, conn net.Conn) error {
	m := &msgQuery{}
	err := json.Unmarshal(buf, m)
	if err != nil {
		s.logError(conn, name, "server can't unmarshal message content!\r\n")
		return err
	}
	s.muChans.Lock()
	resChan, ok := s.activeUsers[m.Receiver]
	if !ok { // if user-receiver is not logged in
		log.Println(name, "- [ERR]: user", m.Receiver, "is not logged in!")
		conn.Write([]byte("EReceiver is not logged in!\r\n"))
		s.muChans.Unlock()
		return nil
	}
	m.Receiver = login
	msg, _ := json.Marshal(m)

	resChan.conn.Write([]byte("M"))
	resChan.conn.Write(msg) // error check
	resChan.conn.Write([]byte("\n"))

	s.muChans.Unlock()
	conn.Write([]byte("OMessage was sent\r\n"))
	return nil
}

func (s *Server) streamMessage(login string, buf []byte, conn net.Conn) {
	s.muChans.Lock()
	for key, value := range s.activeUsers {
		if key == login {
			continue
		}
		value.conn.Write([]byte("M"))
		value.conn.Write(buf) // error check here?
	}
	s.muChans.Unlock()
	conn.Write([]byte("OMessage was streamed\r\n"))
}

func (s *Server) handleCalc(name string, buf []byte, dataChan chan string, conn net.Conn) error {
	c := &calcQuery{}
	err := json.Unmarshal(buf, c) // get params
	if err != nil {
		s.logError(conn, name, "server can't unmarshal message content!\r\n")
		return err
	}

	s.muMap.Lock()
	handler, ok := s.funcMap[c.Function] // find func
	if !ok {
		s.muMap.Unlock()
		log.Println(name, "- [ERR]: can't find function", c.Function)
		conn.Write([]byte("EThis function wasn't registered on server!\r\n"))
		return nil
	}
	s.muChans.Lock() // mutex here, so nobody can't rebind outChan until operation is done
	s.activeUsers[handler].chans.out = &dataChan
	sendChan := s.activeUsers[handler].chans.in
	s.muMap.Unlock()

	*sendChan <- /*"C" + */ string(c.Data) + "\n" // send params to func holder
	// TODO timeout here
	// select time.After
	answer := <-dataChan // get answer
	s.muChans.Unlock()

	log.Println(name, "- [OK]: operation was done")
	conn.Write([]byte(answer)) // send answer to conn with 'D' header
	return nil
}

func (s *Server) handleReady(conn net.Conn, dataChan chan string, resultChan chan []byte, scanner bufio.Reader, u *user) {
	for {
		data := <-dataChan // get values from chan
		conn.Write([]byte("C" + data))
		// timeout?
		ans := <-resultChan
		*u.chans.out <- "D" + string(ans) // send ans to out chan
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	name := conn.RemoteAddr().String()
	log.Println(name, "- [OK]: connected")

	scanner := bufio.NewReader(conn)
	dataChan := make(chan string)
	resultChan := make(chan []byte)
	u := &user{
		conn:  conn,
		chans: &inOutChans{in: &dataChan, out: nil},
	}
	var login string
	ready := false
	defer s.closeConn(conn, &login)
LOOP:
	for {
		msgType, buf, err := s.readMsg(name, *scanner, conn)
		if err != nil {
			break
		}
		log.Printf("%s - type: %s, data: %s\n", name, string(msgType), filterNewLines(string(buf)))

		switch msgType {
		case 'D': // DONE
			resultChan <- buf

		case 'U': // SIGN UP
			log.Println(name, "- [OK]: UP request")
			if err := s.signUpUser(name, buf, conn); err != nil {
				break LOOP
			}

		case 'I': // SIGN IN
			log.Println(name, "- [OK]: IN request")
			if l, err := s.signInUser(name, buf, u, conn); err != nil {
				break LOOP
			} else {
				login = l
			}

		case 'M': // MESSAGE
			if ok := s.checkLogin(login, name, "MSG", conn); !ok {
				break LOOP
			}
			if err := s.sendMessage(name, login, buf, conn); err != nil {
				break LOOP
			}

		case 'S': // STREAM
			if ok := s.checkLogin(login, name, "STR", conn); !ok {
				break LOOP
			}
			s.streamMessage(login, buf, conn)

		case 'C': // CALC - conn asks to calculate calcQuery.Function with parameters stored in calcQuery.Data
			if ok := s.checkLogin(login, name, "CALC", conn); !ok {
				break LOOP
			}
			if err := s.handleCalc(name, buf, dataChan, conn); err != nil {
				break LOOP
			}

		case 'P': // POST - conn tries to declare it's postQuery.Function on server
			if ok := s.checkLogin(login, name, "POST", conn); !ok {
				break LOOP
			}
			if err := s.handlePost(conn, login, buf); err != nil {
				if err.Error() != "function is already exists in map" {
					break LOOP
				}
			}

		case 'R': // READY - conn is now ready to execute others CALC requests
			if ok := s.checkLogin(login, name, "READY", conn); !ok {
				break LOOP
			}

			if ready {
				continue LOOP
			}
			ready = true

			go s.handleReady(conn, dataChan, resultChan, *scanner, u)

		default:
			s.logError(conn, name, "wrong message type!\r\n")
		}
	}
}
