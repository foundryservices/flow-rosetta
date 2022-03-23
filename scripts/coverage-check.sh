#!/bin/bash
# ARG1 = Minimun coverage percent
# ARG2 = Actualy coverage percent as string
PERCENT=$(echo $2 | cut -d'.' -f 1)
PERCENT="${PERCENT//[$'\t\r\n%']}"
echo $2
if [ "$PERCENT" -ge "$1" ]
then
	echo "PASS"
	exit 0
else
	echo "FAIL: Minimum coverage is" $1 "%, you had" $2
	exit 1
fi
