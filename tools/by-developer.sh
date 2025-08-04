#!/usr/bin/env bash

# AUTHOR="foo@bar.com"
# SINCE_DATE="2022-01-01"
function contributions {
  git log --no-merges --since ${SINCE_DATE} --author "${AUTHOR}"  --numstat |\
    grep -v "vendor" |\
    grep -Pv "Date:|insertion|deletion|file|Bin|\.svg|\.drawio|generated|yaml|\.json|html|go\.sum|\.pb\.go|\.pb-c|\=\>" | sort -k3 |\
    grep -P "^\d+\t\d+" |\
    awk 'BEGIN{total=0}{total+=$1+$2}END{print total}'
}

# AUTHOR="foo@bar.com"
# SINCE_DATE="2022-01-01"
# UNTIL_DATE="2023-01-01"
function contributions-period {
  echo $AUTHOR $SINCE_DATE $UNTIL_DATE
  git log --no-merges --since ${SINCE_DATE} --until ${UNTIL_DATE} --author "${AUTHOR}"  --numstat |\
    grep -v "vendor" |\
    grep -Pv "Date:|insertion|deletion|file|Bin|\.svg|\.drawio|generated|yaml|\.json|html|go\.sum|\.pb\.go|\.pb-c|\=\>" | sort -k3 |\
    grep -P "^\d+\t\d+" |\
    awk 'BEGIN{total=0}{total+=$1}END{print total}'
  git log --no-merges --since ${SINCE_DATE} --until ${UNTIL_DATE} --author "${AUTHOR}"  --numstat |\
    grep -v "vendor" |\
    grep -Pv "Date:|insertion|deletion|file|Bin|\.svg|\.drawio|generated|yaml|\.json|html|go\.sum|\.pb\.go|\.pb-c|\=\>" | sort -k3 |\
    grep -P "^\d+\t\d+" |\
    awk 'BEGIN{total=0}{total+=$2}END{print total}'
  git log --no-merges --since ${SINCE_DATE} --until ${UNTIL_DATE} --author "${AUTHOR}"  --numstat |\
    grep -v "vendor" |\
    grep -Pv "Date:|insertion|deletion|file|Bin|\.svg|\.drawio|generated|yaml|\.json|html|go\.sum|\.pb\.go|\.pb-c|\=\>" | sort -k3 |\
    grep -P "^\d+\t\d+" |\
    awk 'BEGIN{total=0}{total+=$1+$2}END{print total}'
}


contributions-period
