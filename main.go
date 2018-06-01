package main

import (
	"io"
	"log"
	"net"
	"os"
	"time"
)

const (
	logPath = "./logs/"
	address = "8080"
)

func setupLogger() *os.File {
	logFile, err := os.OpenFile(logPath+time.Now().String()+".log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("server - [ERR]: Cannot open log file: %v\n", err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
	return logFile
}

func main() {
	logFile := setupLogger()
	defer logFile.Close()
	log.Println("Starting server...")

	listner, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}

	server := NewServer()
	for {
		conn, err := listner.Accept()
		if err != nil {
			panic(err)
		}
		go server.handleConnection(conn)
	}
}
