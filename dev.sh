#!/bin/bash
############################## DEV_SCRIPT_MARKER ##############################
# This script is used to document and run recurring tasks in development.     #
#                                                                             #
# You can run your tasks using the script `./dev some-task`.                  #
# You can install the Sandstorm Dev Script Runner and run your tasks from any #
# nested folder using `dev some-task`.                                        #
# https://github.com/sandstorm/Sandstorm.DevScriptRunner                      #
###############################################################################

set -e

######### TASKS #########

function run-tests() {
  go mod tidy
  go test -v github.com/sandstorm/caddy-nats-bridge/...
}


function run-example() {
  # currently deactivated. we might need it later sometime.
  exit 1

  killall nats || true
  killall caddyexample  || true

  nats subscribe '>' &
  ./caddyexample run --config Caddyfile &
  sleep 1


  # curl -H "Foo: bar" https://localhost/test/foo?x=b

  #mkfile -n 2m temp_2m_file
  mkfile -n 500k temp_500k_file
  #curl -F data=@temp_2m_file https://localhost/test/foo?x=b
  #curl -H "Transfer-Encoding: chunked" -F data=@temp_2m_file https://localhost/test/foo?x=b
  #curl -F "data=@temp_500k_file" https://localhost/test/foo?x=b


  killall nats
  killall caddyexample
}


####### Utilities #######

_log_success() {
  printf "\033[0;32m%s\033[0m\n" "${1}"
}
_log_warning() {
  printf "\033[1;33m%s\033[0m\n" "${1}"
}
_log_error() {
  printf "\033[0;31m%s\033[0m\n" "${1}"
}

# THIS NEEDS TO BE LAST!!!
# this will run your tasks
"$@"
