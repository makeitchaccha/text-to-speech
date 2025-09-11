#!/bin/sh
set -e

# Allow the container to be started with a different command
if [ "$1" = 'migrate' ]; then
    shift
    # run goose with the remaining arguments
    exec goose "$@"
fi

# If the command is not 'migrate', run the main application
exec /bin/bot "$@"
