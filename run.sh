#!/bin/bash
export PATH="/usr/local/go/bin:/home/joke/go/bin:$PATH"
cd /home/joke/Work/scs
exec "$@"

# ./scs -i test-100000.txt -o test-1000000-2.scs --profile && go tool pprof -http=:8080 cpu.prof
