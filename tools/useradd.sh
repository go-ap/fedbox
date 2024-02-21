#!/usr/bin/expect -f
# Special thanks to github.com/squash for providing this script in the course of solving
# https://github.com/mariusor/go-littr/issues/38#issuecomment-658800183

set name [lindex $argv 0]
set pass [lindex $argv 1]

spawn ./bin/fedboxctl ap actor add ${name}
expect "${name}'s pw: "
send "${pass}\r"
expect "pw again: "
send "${pass}\r"
expect eof

