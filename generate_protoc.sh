#!/usr/bin/env bash

python3 -m grpc_tools.protoc -I. --python_out=client --grpc_python_out=client medical_test.proto

protoc -I. medical_test.proto --go_out=plugins=grpc:server/medical_test

