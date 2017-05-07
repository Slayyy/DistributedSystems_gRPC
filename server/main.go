package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	med "./medical_test"
)

const (
	dbFileName = "db.json"
)

type db struct {
	Patients    map[int64]*med.Patient
	Doctors     map[int64]*med.Doctor
	Technicians map[int64]*med.Technican
	MedTests    map[int64]*med.MedicalTest
}

func (d *db) LoadFromFile(filePath string) error {
	blob, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(blob, d)
}

func (d *db) SaveToFile(filePath string) error {
	blob, err := json.Marshal(d)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filePath, blob, 0666)
}

func main() {
	db := db{}
	err := db.LoadFromFile(dbFileName)
	if err != nil {
		log.Fatal(err)
	}

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		err := db.SaveToFile(dbFileName)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	const port = ":30000"
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal(err)
	}
	server := grpc.NewServer()

	reflection.Register(server)
	server.Serve(listener)
}

type patientServer struct {
	*db
}
