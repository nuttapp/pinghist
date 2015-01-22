#!/bin/sh
# Run all tests...

# This oneliner monstrosity removes extra lines and vertical junk from the output of GoConvey. 
# It selectively removes colorized outout, and adds it back when a test fails (red)
unset REPORTTIME 

# Run unit tests
go test -v -run unit ./... | sed "s/.* assertion.*/[0m/" | grep -v -E "^($|\?|PASS|ok)" | sed 's/^\[.*//g' | egrep -v '^[[:space:]]*$' |sed 's/===/\
/g'  | sed "s/---//g" | sed "s/Failures:/[38;5;160mFAIL/g" | sed "s/RUN/[0mRUN/g"

# Run Integration tests
go test -v -run inte ./... | sed "s/.* assertion.*/[0m/" | grep -v -E "^($|\?|PASS|ok)" | sed 's/^\[.*//g' | egrep -v '^[[:space:]]*$' |sed 's/===/\
/g'  | sed "s/---//g" | sed "s/Failures:/[38;5;160mFAIL/g" | sed "s/RUN/[0mRUN/g"


