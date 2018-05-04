package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
)

type MsgQuery struct {
	Receiver string `json:"rec"`
	Message  string `json:"msg"`
}

func Login(name string, conn net.Conn, reader bufio.Reader) {
	conn.Write([]byte("I{\"login\":\"" + name + "\",\"pass\":\"lol\"}\n"))
	msgType, err := reader.ReadByte()
	message, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	if msgType == 'E' {
		panic(string(message))
	}
	fmt.Println(string(message))
}

func SendMessages(conn net.Conn) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n[OUT] Enter receiver: ")
		res, _ := reader.ReadString('\n')
		fmt.Print("\n[OUT] Enter message: ")
		msg, _ := reader.ReadString('\n')
		m := &MsgQuery{
			Receiver: strings.TrimSpace(res),
			Message:  strings.TrimSpace(msg),
		}
		byt, err := json.Marshal(m)
		if err != nil {
			panic(err)
		}
		conn.Write([]byte("M"))
		conn.Write(byt)
		conn.Write([]byte("\n")) // always add \n at the end!
	}
}

func StreamMessages(login string, conn net.Conn) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n[OUT] Enter message: ")
		msg, _ := reader.ReadString('\n')
		m := &MsgQuery{
			Receiver: login, // sender here
			Message:  strings.TrimSpace(msg),
		}
		byt, err := json.Marshal(m)
		if err != nil {
			panic(err)
		}
		conn.Write([]byte("S"))
		conn.Write(byt)
		conn.Write([]byte("\n")) // always add \n at the end!
	}
}

func ReadMessages(reader bufio.Reader) {
	for {
		msgType, err := reader.ReadByte()
		msg, err := reader.ReadBytes('\n')
		if err != nil {
			panic(err)
		}
		if msgType == 'E' {
			fmt.Println("\n[ERR]", string(msg))
		} else if msgType == 'M' {
			m := &MsgQuery{}
			err = json.Unmarshal(msg, m)
			fmt.Printf("\n[IN] From: %s; Msg: %s\n", m.Receiver, m.Message)
		} else {
			fmt.Println("\n[IN] Unexpected message type!", string(msg))
		}
	}
}
