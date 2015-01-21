#!/bin/sh
unset REPORTTIME 

go test -v -run Unit ./... | sed "s/.* assertion.*/[0m/" | grep -v -E "^($|\?|PASS|ok)" | sed 's/^\[.*//g' | egrep -v '^[[:space:]]*$' |sed 's/===/\
/g'  | sed "s/---//g" | sed "s/Failures:/[38;5;160mFAIL/g" | sed "s/RUN/[0mRUN/g"

go test -v -run Inte ./... | sed "s/.* assertion.*/[0m/" | grep -v -E "^($|\?|PASS|ok)" | sed 's/^\[.*//g' | egrep -v '^[[:space:]]*$' |sed 's/===/\
/g'  | sed "s/---//g" | sed "s/Failures:/[38;5;160mFAIL/g" | sed "s/RUN/[0mRUN/g"


