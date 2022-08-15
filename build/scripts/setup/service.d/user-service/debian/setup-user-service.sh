#!/bin/bash

set -e

## base variables
BUILD_PATH=${1:?missing build path}
SOURCE_ROOT=${BUILD_PATH}/sysroot

APP_NAME="casaos-user-service"
APP_NAME_SHORT="user-service"

MIGRATION_SCRIPT_PATH=${BUILD_PATH}/scripts/migration/script.d/01-migrate-${APP_NAME_SHORT}.sh

$SHELL "${MIGRATION_SCRIPT_PATH}"

# copy sysroot over

cp -rv "${SOURCE_ROOT}"/* /

# copy config files
CONF_PATH=/etc/casaos
CONF_FILE=${CONF_PATH}/${APP_NAME_SHORT}.conf
CONF_FILE_SAMPLE=${CONF_PATH}/${APP_NAME_SHORT}.conf.sample

if [ ! -f "${CONF_FILE}" ]; then \
    echo "Initializing config file..."
    cp -v "${CONF_FILE_SAMPLE}" "${CONF_FILE}"; \
fi

# enable and start service

echo "Enabling service..."
systemctl enable --force --no-ask-password "${APP_NAME}.service"

echo "Starting service..."
systemctl start --force --no-ask-password "${APP_NAME}.service"
