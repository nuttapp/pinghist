#!/bin/sh
# Run all tests...
# The sed/grep monstrosity removes extra lines and vertical junk from the verbose output of GoConvey. 
# It selectively removes colorized outout, but adds it back when a test fails.
unset REPORTTIME 

# go test -v ./... -run dal_integ

# # Run unit tests
go test -v -run unit ./... | sed "s/.* assertion.*/[0m/" | grep -v -E "^($|\?|PASS|ok)" | sed 's/^\[.*//g' | egrep -v '^[[:space:]]*$' |sed 's/===/\
/g'  | sed "s/---//g" | sed "s/Failures:/[38;5;160mFAIL/g" | sed "s/RUN/[0mRUN/g"
#
# # Run Integration tests
go test -v -run inte ./... | sed "s/.* assertion.*/[0m/" | grep -v -E "^($|\?|PASS|ok)" | sed 's/^\[.*//g' | egrep -v '^[[:space:]]*$' |sed 's/===/\
/g'  | sed "s/---//g" | sed "s/Failures:/[38;5;160mFAIL/g" | sed "s/RUN/[0mRUN/g"


