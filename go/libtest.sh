#!/bin/bash

# Assumes GOPATH only has one path for your project
PROJECT_PATH=$GOPATH

# Pick random ports between [10000, 20000)
STORAGE_PORT=$(((RANDOM % 10000) + 10000))
LIB_PORT=$(((RANDOM % 10000) + 10000))

# Build libtest
cd ${PROJECT_PATH}/src/P2-f12/official/libtest
go build
cd - > /dev/null

# Start storage server
${PROJECT_PATH}/bin/linux_amd64/storageserver -port=${STORAGE_PORT} 2> /dev/null &
STORAGE_SERVER_PID=$!
sleep 5

# Start libtest
${PROJECT_PATH}/src/P2-f12/official/libtest/libtest -port=${LIB_PORT} "localhost:${STORAGE_PORT}"

# Kill storage server
kill -9 ${STORAGE_SERVER_PID}
wait ${STORAGE_SERVER_PID} 2> /dev/null
