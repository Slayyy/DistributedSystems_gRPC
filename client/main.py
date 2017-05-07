#!/usr/bin/env python3
import sys

import fire

import grpc
import medical_test_pb2
import medical_test_pb2_grpc


channel = grpc.insecure_channel('localhost:30000')

patientStub = medical_test_pb2_grpc.PatientServiceStub(channel)
doctorStub = medical_test_pb2_grpc.DoctorServiceStub(channel)
technicianStub = medical_test_pb2_grpc.TechnicianServiceStub(channel)
dbStub = medical_test_pb2_grpc.DbServiceStub(channel)


def print_allowed_methods_and_exit(allowed_methods):
    print("Allowed methods: {}.".format(", ".join(allowed_methods)))
    sys.Exit(1)


def print_med_test(result):
    print("Test: {}\nPatient: {}\nDoctor: {}\nTechnician: {}".format(
            result.id,
            result.patient.person.id,
            result.doctor.person.id,
            result.technician.person.id))

    for k, v in result.results.items():
        print("\t{}: {}".format(k, v))
    print("\n")


def print_person(p):
    print("ID: {}\nName:{}\n\n".format(p.id, p.name))


class Client(object):
    def patient_fetch_all_results(self, id):
        queryId = medical_test_pb2.Id()
        queryId.id = id
        for result in patientStub.FetchAllResults(queryId):
            print_med_test(result)

    def doctor_fetch_all_results(self):
        empty = medical_test_pb2.Empty()
        for result in doctorStub.FetchAllResults(empty):
            print_med_test(result)

    def doctor_fetch_results(self, id):
        query_id = medical_test_pb2.Id()
        query_id.id = id
        result = doctorStub.FetchResults(query_id)
        print_med_test(result)

    def doctor_fetch_on_id_parity(self, isEven):
        query_id = medical_test_pb2.FilterEvenId()
        query_id.isEven = True if isEven.lower() == "true" else False

        for result in doctorStub.FetchOnIdParity(query_id):
            print_med_test(result)

    def technician_add_results(self, technician_id, doctor_id, patient_id,
                               *results):
        new_medtest_rq = medical_test_pb2.NewMedTestRequest()
        new_medtest_rq.technician = technician_id
        new_medtest_rq.doctor = doctor_id
        new_medtest_rq.patient = patient_id
        for result in results:
            key, value = result.split(":")
            new_medtest_rq.results[key] = value

        result = technicianStub.AddResults(new_medtest_rq)
        status = result.status
        print("{}".format(status))

    def db_new_patient(self, name):
        person = medical_test_pb2.Person()
        person.name = name
        print(dbStub.NewPatient(person))

    def db_new_doctor(self, name):
        person = medical_test_pb2.Person()
        person.name = name
        print(dbStub.NewDoctor(person))

    def db_new_technician(self, name):
        person = medical_test_pb2.Person()
        person.name = name
        print(dbStub.NewTechnician(person))

    def db_fetch_patients(self):
        empty = medical_test_pb2.Empty()
        for result in dbStub.FetchPatients(empty):
            print_person(result.person)

    def db_fetch_doctors(self):
        empty = medical_test_pb2.Empty()
        for result in dbStub.FetchDoctors(empty):
            print_person(result.person)

    def db_fetch_technicians(self):
        empty = medical_test_pb2.Empty()
        for result in dbStub.FetchTechnicians(empty):
            print_person(result.person)


if __name__ == '__main__':
    fire.Fire(Client)
