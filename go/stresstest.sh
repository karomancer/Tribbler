#!/bin/bash

# Assumes GOPATH only has one path for your project
PROJECT_PATH=$GOPATH

# Pick random port between [10000, 20000)
STORAGE_PORT=$(((RANDOM % 10000) + 10000))

# Build tribserver
cd ${PROJECT_PATH}/src/P2-f12/official/tribserver
go build
cd - > /dev/null

# Build storageserver
cd ${PROJECT_PATH}/src/P2-f12/official/storageserver
go build
cd - > /dev/null

# Build stressclient
cd ${PROJECT_PATH}/src/P2-f12/official/stressclient
go build
cd - > /dev/null

function startStorageServers {
    N=${#STORAGE_ID[@]}
    # Start master storage server
    ${PROJECT_PATH}/src/P2-f12/official/storageserver/storageserver -N=${N} -id=${STORAGE_ID[0]} -port=${STORAGE_PORT} &> /dev/null &
    STORAGE_SERVER_PID[0]=$!
    # Start slave storage servers
    if [ "$N" -gt 1 ]
    then
        for i in `seq 1 $((N - 1))`
        do
            ${PROJECT_PATH}/src/P2-f12/official/storageserver/storageserver -id=${STORAGE_ID[$i]} -master="localhost:${STORAGE_PORT}" &> ssOUT.txt &
            STORAGE_SERVER_PID[$i]=$!
        done
    fi
    sleep 5
}

function stopStorageServers {
    N=${#STORAGE_ID[@]}
    for i in `seq 0 $((N - 1))`
    do
        kill -9 ${STORAGE_SERVER_PID[$i]}
        wait ${STORAGE_SERVER_PID[$i]} 2> /dev/null
    done
}

function startTribServers {
    for i in `seq 0 $((M - 1))`
    do
        # Pick random port between [10000, 20000)
        TRIB_PORT[$i]=$(((RANDOM % 10000) + 10000))
        ${PROJECT_PATH}/src/P2-f12/official/tribserver/tribserver -port=${TRIB_PORT[$i]} "localhost:${STORAGE_PORT}" &> tsOUT.txt &
        TRIB_SERVER_PID[$i]=$!
    done
    sleep 5
}

function stopTribServers {
    for i in `seq 0 $((M - 1))`
    do
        kill -9 ${TRIB_SERVER_PID[$i]}
        wait ${TRIB_SERVER_PID[$i]} 2> /dev/null
    done
}

function testStress {
    startStorageServers
    startTribServers
    # Start stress clients
    C=0
    K=${#CLIENT_COUNT[@]}
    for USER in `seq 0 $((K - 1))`
    do
        for CLIENT in `seq 0 $((CLIENT_COUNT[$USER] - 1))`
        do
            ${PROJECT_PATH}/src/P2-f12/official/stressclient/stressclient -port=${TRIB_PORT[$((C % M))]} -clientId=${CLIENT} ${USER} ${K} & 
            STRESS_CLIENT_PID[$C]=$!
            # Setup background thread to kill client upon timeout
            sleep ${TIMEOUT} && kill -9 ${STRESS_CLIENT_PID[$C]} &> stressOUT.txt &
            C=$((C + 1))
        done
    done

    # Check exit status
    FAIL=0
    for i in `seq 0 $((C - 1))`
    do
        wait ${STRESS_CLIENT_PID[$i]} 2> /dev/null
        if [ "$?" -ne 7 ]
        then
            FAIL=$((FAIL + 1))
        fi
    done
    if [ "$FAIL" -eq 0 ]
    then
        echo "PASS"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo "FAIL: ${FAIL} clients failed"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
    stopTribServers
    stopStorageServers
}

# Testing Single client, single tribserver, single storageserver
function testStressSingleClientSingleTribSingleStorage {
    echo "Starting testStressSingleClientSingleTribSingleStorage:"
    STORAGE_ID=('0')
    M=1
    CLIENT_COUNT=('1')
    TIMEOUT=10
    testStress
}

# Testing Single client, single tribserver, Multiple storageserver
function testStressSingleClientSingleTribMultipleStorage {
    echo "Starting testStressSingleClientSingleTribMultipleStorage"
    STORAGE_ID=('0' '0' '0')
    M=1
    CLIENT_COUNT=('1')
    TIMEOUT=10
    testStress
}

# Testing Multiple client, single tribserver, single storageserver
function testStressMultipleClientSingleTribSingleStorage {
    echo "Starting testStressMultipleClientSingleTribSingleStorage"
    STORAGE_ID=('0')
    M=1
    CLIENT_COUNT=('1' '1' '1')
    TIMEOUT=10
    testStress
}

# Testing Multiple client, single tribserver, multiple storageserver
function testStressMultipleClientSingleTribMultipleStorage {
    echo "Starting testStressMultipleClientSingleTribMultipleStorage"
    STORAGE_ID=('0' '0' '0' '0' '0' '0')
    M=1
    CLIENT_COUNT=('1' '1' '1')
    TIMEOUT=10
    testStress
}

# Testing Multiple client, multiple tribserver, single storageserver
function testStressMultipleClientMultipleTribSingleStorage {
    echo "Starting testStressMultipleClientMultipleTribSingleStorage"
    STORAGE_ID=('0')
    M=2
    CLIENT_COUNT=('1' '1')
    TIMEOUT=30
    testStress
}

# Testing Multiple client, multiple tribserver, multiple storageserver
function testStressMultipleClientMultipleTribMultipleStorage {
    echo "Starting testStressMultipleClientMultipleTribMultipleStorage"
    STORAGE_ID=('0' '0' '0' '0' '0' '0' '0')
    M=3
    CLIENT_COUNT=('1' '1' '1')
    TIMEOUT=30
    testStress
}

# Testing 2x more clients than tribservers, multiple tribserver, multiple storageserver
function testStressDoubleClientMultipleTribMultipleStorage {
    echo "Starting testStressDoubleClientMultipleTribMultipleStorage"
    STORAGE_ID=('0' '0' '0' '0' '0' '0')
    M=2
    CLIENT_COUNT=('1' '1' '1' '1')
    TIMEOUT=30
    testStress
}


# Testing duplicate users, multiple tribserver, single storageserver
function testStressDupUserMultipleTribSingleStorage {
    echo "Starting testStressDupUserMultipleTribSingleStorage"
    STORAGE_ID=('0')
    M=2
    CLIENT_COUNT=('2')
    TIMEOUT=30
    testStress
}

# Testing duplicate users, multiple tribserver, multiple storageserver
function testStressDupUserMultipleTribMultipleStorage {
    echo "Starting testStressDupUserMultipleTribMultipleStorage"
    STORAGE_ID=('0' '0' '0')
    M=2
    CLIENT_COUNT=('2')
    TIMEOUT=30
    testStress
}

# Run tests
PASS_COUNT=0
FAIL_COUNT=0
testStressSingleClientSingleTribSingleStorage
#testStressSingleClientSingleTribMultipleStorage
#testStressMultipleClientSingleTribSingleStorage
#testStressMultipleClientSingleTribMultipleStorage
#testStressMultipleClientMultipleTribSingleStorage
#testStressMultipleClientMultipleTribMultipleStorage
#testStressDoubleClientMultipleTribMultipleStorage
#testStressDupUserMultipleTribSingleStorage
#testStressDupUserMultipleTribMultipleStorage

echo "Passed (${PASS_COUNT}/$((PASS_COUNT + FAIL_COUNT))) tests"
