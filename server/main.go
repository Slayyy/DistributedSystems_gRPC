package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	med "./medical_test"
)

const (
	dbFileName = "db.json"
)

func toJSON(x interface{}) string {
	blob, _ := json.Marshal(x)
	return string(blob)
}

type db struct {
	Patients    map[int64]*med.Patient
	Doctors     map[int64]*med.Doctor
	Technicians map[int64]*med.Technician
	MedTests    map[int64]*med.MedicalTest

	PersonCounter  int64
	MedTestCounter int64
}

func (d *db) InitEmpty() {
	d.Patients = make(map[int64]*med.Patient)
	d.Doctors = make(map[int64]*med.Doctor)
	d.Technicians = make(map[int64]*med.Technician)
	d.MedTests = make(map[int64]*med.MedicalTest)

	d.PersonCounter = 0
	d.MedTestCounter = 0

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
	db := &db{}
	log.Println("LoadFromFile")
	err := db.LoadFromFile(dbFileName)
	if err != nil {
		log.Println(err)
		log.Println("Initializing empty db")
		db.InitEmpty()
	}
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("SaveToFile")
		err := db.SaveToFile(dbFileName)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	const port = ":30000"
	log.Println("Listen")
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal(err)
	}

	server := grpc.NewServer()

	med.RegisterPatientServiceServer(server, &patientImpl{db: db})
	med.RegisterTechnicianServiceServer(server, &technicianImpl{db: db})
	med.RegisterDoctorServiceServer(server, &doctorImpl{db: db})
	med.RegisterDbServiceServer(server, &dbImpl{db: db})

	reflection.Register(server)
	log.Println("----------START----------")
	server.Serve(listener)
}

type patientImpl struct {
	db *db
}

func (pi *patientImpl) FetchAllResults(id *med.Id, stream med.PatientService_FetchAllResultsServer) error {
	log.Println("FetchAllResults: ", toJSON(id))
	for _, val := range pi.db.MedTests {
		if val.Patient.Person.Id == id.Id {
			err := stream.Send(val)
			if err != nil {
				log.Print(err)
				return err
			}
		}
	}
	return nil
}

type technicianImpl struct {
	db *db
}

func (ti *technicianImpl) AddResults(ctx context.Context, rq *med.NewMedTestRequest) (*med.Status, error) {
	log.Println("AddResults:", toJSON(rq))
	notFounds := []string{}

	doctor := ti.db.Doctors[rq.Doctor]
	if doctor == nil {
		notFounds = append(notFounds, "doctor")
	}

	technician := ti.db.Technicians[rq.Technician]
	if technician == nil {
		notFounds = append(notFounds, "technician")
	}

	patient := ti.db.Patients[rq.Patient]
	if patient == nil {
		notFounds = append(notFounds, "patient")
	}

	if len(notFounds) > 0 {
		message := fmt.Sprintf("Cannot find: %s", strings.Join(notFounds, ", "))
		return &med.Status{Status: message}, nil
	}

	ti.db.MedTestCounter++
	newID := ti.db.MedTestCounter
	ti.db.MedTests[newID] = &med.MedicalTest{
		Id:         newID,
		Doctor:     doctor,
		Patient:    patient,
		Technician: technician,
		Results:    rq.Results,
	}
	return &med.Status{Status: "Created"}, nil
}

type doctorImpl struct {
	db *db
}

func (di *doctorImpl) FetchAllResults(e *med.Empty, stream med.DoctorService_FetchAllResultsServer) error {
	log.Println("FetchAllResults")
	for _, val := range di.db.MedTests {
		err := stream.Send(val)
		if err != nil {
			log.Print(err)
			return err
		}
	}
	return nil
}

func (di *doctorImpl) FetchResults(ctx context.Context, id *med.Id) (*med.MedicalTest, error) {
	results := di.db.MedTests[id.Id]
	if results == nil {
		return nil, errors.New("Not found")
	}
	return results, nil
}

func (di *doctorImpl) FetchOnIdParity(e *med.FilterEvenId, stream med.DoctorService_FetchOnIdParityServer) error {
	log.Println("FetchOnIdParity:", e.IsEven)

	var n int64
	if e.IsEven {
		n = 0
	} else {
		n = 1
	}
	for _, val := range di.db.MedTests {
		if val.Id%2 != n {
			continue
		}
		err := stream.Send(val)
		if err != nil {
			log.Print(err)
			return err
		}
	}
	return nil
}

type dbImpl struct {
	db *db
}

func (d *dbImpl) NewPatient(ctx context.Context, person *med.Person) (*med.Patient, error) {
	log.Println("NewPatient:", toJSON(person))
	d.db.PersonCounter++
	newID := d.db.PersonCounter
	person.Id = newID
	patient := &med.Patient{person}
	d.db.Patients[newID] = patient
	return patient, nil
}

func (d *dbImpl) NewDoctor(ctx context.Context, person *med.Person) (*med.Doctor, error) {
	log.Println("NewDoctor:", toJSON(person))
	d.db.PersonCounter++
	newID := d.db.PersonCounter
	person.Id = newID
	doctor := &med.Doctor{person}
	d.db.Doctors[newID] = doctor
	return doctor, nil
}

func (d *dbImpl) NewTechnician(ctx context.Context, person *med.Person) (*med.Technician, error) {
	log.Println("NewTechnician:", toJSON(person))
	d.db.PersonCounter++
	newID := d.db.PersonCounter
	person.Id = newID
	tech := &med.Technician{person}
	d.db.Technicians[newID] = tech
	return tech, nil
}

func (d *dbImpl) FetchPatients(e *med.Empty, stream med.DbService_FetchPatientsServer) error {
	log.Println("FetchPatients")
	for _, val := range d.db.Patients {
		err := stream.Send(val)
		if err != nil {
			log.Print(err)
			return err
		}
	}
	return nil
}

func (d *dbImpl) FetchDoctors(e *med.Empty, stream med.DbService_FetchDoctorsServer) error {
	log.Println("FetchDoctors")
	for _, val := range d.db.Doctors {
		err := stream.Send(val)
		if err != nil {
			log.Print(err)
			return err
		}
	}
	return nil
}
func (d *dbImpl) FetchTechnicians(e *med.Empty, stream med.DbService_FetchTechniciansServer) error {
	log.Println("FetchTechnicians")
	for _, val := range d.db.Technicians {
		err := stream.Send(val)
		if err != nil {
			log.Print(err)
			return err
		}
	}
	return nil
}
