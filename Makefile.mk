# project service name
NAME := workout

# local database connection settings
TEST_PGDATABASE ?= test-workout

PGDATABASE ?= workout
PGHOST ?= localhost
PGPORT ?= 5432
PGUSER ?= postgres
PGPASSWORD ?= postgres

# add -race to GOFLAGS if RACE=1
RACE=0
