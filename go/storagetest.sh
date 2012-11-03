#!/bin/bash

# Assumes GOPATH only has one path for your project
PROJECT_PATH=$GOPATH

# Pick random ports between [10000, 20000)
STORAGE_PORT=$(((RANDOM % 10000) + 10000))
TESTER_PORT=$(((RANDOM % 10000) + 10000))

# Build storagetest
cd ${PROJECT_PATH}/src/P2-f12/official/storagetest
go build
cd - > /dev/null

# Build storageserver
cd ${PROJECT_PATH}/src/P2-f12/official/storageserver
go build
cd - > /dev/null

#
# start btest
#
# Start storage server
${PROJECT_PATH}/src/P2-f12/official/storageserver/storageserver -port=${STORAGE_PORT} 2> /dev/null &
STORAGE_SERVER_PID=$!
sleep 5

# Start storagetest
${PROJECT_PATH}/src/P2-f12/official/storagetest/storagetest -port=${TESTER_PORT} -type=2 "localhost:${STORAGE_PORT}"

# Kill storage server
kill -9 ${STORAGE_SERVER_PID}
wait ${STORAGE_SERVER_PID} 2> /dev/null

#
# start jtest
#
# Start storage server
${PROJECT_PATH}/src/P2-f12/official/storageserver/storageserver -port=${STORAGE_PORT} -N=2 -id=900 2> /dev/null &
STORAGE_SERVER_PID=$!
sleep 5

# Start storagetest
${PROJECT_PATH}/src/P2-f12/official/storagetest/storagetest -port=${TESTER_PORT} -type=1 -N=2 -id=800 "localhost:${STORAGE_PORT}"

# Kill storage server
kill -9 ${STORAGE_SERVER_PID}
wait ${STORAGE_SERVER_PID} 2> /dev/null
