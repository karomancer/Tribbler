#!/bin/bash

# Assumes GOPATH only has one path for your project
PROJECT_PATH=$GOPATH

# Pick random port between [10000, 20000)
STORAGE_PORT=$(((RANDOM % 10000) + 10000))

# Build libclient
cd ${PROJECT_PATH}/src/P2-f12/official/libclient
go build
cd - > /dev/null

function startStorageServers {
    N=${#STORAGE_ID[@]}
    # Start master storage server
    ${PROJECT_PATH}/bin/linux_amd64/storageserver -N=${N} -id=${STORAGE_ID[0]} -port=${STORAGE_PORT} 2> /dev/null &
    STORAGE_SERVER_PID[0]=$!
    # Start slave storage servers
    if [ "$N" -gt 1 ]
    then
        for i in `seq 1 $((N-1))`
        do
            ${PROJECT_PATH}/bin/linux_amd64/storageserver -N=${N} -id=${STORAGE_ID[$i]} -master="localhost:${STORAGE_PORT}" 2> /dev/null &
            STORAGE_SERVER_PID[$i]=$!
        done
    fi
    sleep 5
}

function stopStorageServers {
    N=${#STORAGE_ID[@]}
    for i in `seq 0 $((N-1))`
    do
        kill -9 ${STORAGE_SERVER_PID[$i]}
        wait ${STORAGE_SERVER_PID[$i]} 2> /dev/null
    done
}

function testRouting {
    startStorageServers
    for KEY in "${KEYS[@]}"
    do
        ${PROJECT_PATH}/src/P2-f12/official/libclient/libclient -port=${STORAGE_PORT} p ${KEY} value > /dev/null
        PASS=`${PROJECT_PATH}/src/P2-f12/official/libclient/libclient -port=${STORAGE_PORT} g ${KEY} | grep value | wc -l`
        if [ "$PASS" -ne 1 ]
        then
            break
        fi
    done
    if [ "$PASS" -eq 1 ]
    then
        echo "PASS"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo "FAIL: incorrect value"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
    stopStorageServers
}

# Testing delayed start
function testDelayedStart {
    echo "Starting testDelayedStart:"

    # Start master storage server
    ${PROJECT_PATH}/bin/linux_amd64/storageserver -N=2 -port=${STORAGE_PORT} 2> /dev/null &
    STORAGE_SERVER_PID1=$!
    sleep 5

    # Run libclient
    ${PROJECT_PATH}/src/P2-f12/official/libclient/libclient -port=${STORAGE_PORT} p "key:" value &> /dev/null &
    sleep 3

    # Start second storage server
    ${PROJECT_PATH}/bin/linux_amd64/storageserver -N=2 -master="localhost:${STORAGE_PORT}" 2> /dev/null &
    STORAGE_SERVER_PID2=$!
    sleep 5

    # Run libclient
    PASS=`${PROJECT_PATH}/src/P2-f12/official/libclient/libclient -port=${STORAGE_PORT} g "key:" | grep value | wc -l`
    if [ "$PASS" -eq 1 ]
    then
        echo "PASS"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo "FAIL: incorrect value"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi

    # Kill storage servers
    kill -9 ${STORAGE_SERVER_PID1}
    kill -9 ${STORAGE_SERVER_PID2}
    wait ${STORAGE_SERVER_PID1} 2> /dev/null
    wait ${STORAGE_SERVER_PID2} 2> /dev/null
}

# Testing routing general
function testRoutingGeneral {
    echo "Starting testRoutingGeneral:"
    STORAGE_ID=('3000000000' '4000000000' '2000000000')
    KEYS=('bubble:' 'insertion:' 'merge:' 'heap:' 'quick:' 'radix:')
    testRouting
}

# Testing routing wraparound
function testRoutingWraparound {
    echo "Starting testRoutingWraparound:"
    STORAGE_ID=('2000000000' '2500000000' '3000000000')
    KEYS=('bubble:' 'insertion:' 'merge:' 'heap:' 'quick:' 'radix:')
    testRouting
}

# Testing routing equal
function testRoutingEqual {
    echo "Starting testRoutingEqual:"
    STORAGE_ID=('3835649095' '1581790440' '2373009399' '3448274451' '1666346102' '2548238361')
    KEYS=('bubble:' 'insertion:' 'merge:' 'heap:' 'quick:' 'radix:')
    testRouting
}

# Run tests
PASS_COUNT=0
FAIL_COUNT=0
testDelayedStart
testRoutingGeneral
testRoutingWraparound
testRoutingEqual

echo "Passed (${PASS_COUNT}/$((PASS_COUNT + FAIL_COUNT))) tests"
