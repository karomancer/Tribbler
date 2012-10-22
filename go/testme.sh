#!/bin/sh

# Not an official set of tests, just something to
# help you test out a bit.

./bin/tribclient uc dga
./bin/tribclient uc bryant
./bin/tribclient uc rstarzl
./bin/tribclient uc imoraru

echo "Part 1 of 5"

./bin/tribclient sl dga
./bin/tribclient sa dga bryant
./bin/tribclient sl dga
./bin/tribclient sa dga imoraru
./bin/tribclient sl dga

echo "Part 2 of 5"

./bin/tribclient tl dga
./bin/tribclient tp dga "First post"
./bin/tribclient tl dga
./bin/tribclient tp dga "Second post"
./bin/tribclient tl dga

echo "Part 3 of 5"

./bin/tribclient sa bryant imoraru
./bin/tribclient sa bryant dga
./bin/tribclient tp imoraru "Iulian's first post"
./bin/tribclient tp imoraru "Iulian's second post"
./bin/tribclient ts bryant

echo "Part 4 of 5"

./bin/tribclient sr bryant imoraru
./bin/tribclient sl bryant
./bin/tribclient ts bryant

echo "Part 5 of 5"

echo "This next remove should fail"
./bin/tribclient sr bryant imoraru
