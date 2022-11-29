#!/bin/bash

set -e

readonly CASA_EXEC=casaos-user-service
readonly CASA_SERVICE=casaos-user-service.service

CASA_SERVICE_PATH=$(systemctl show ${CASA_SERVICE} --no-pager  --property FragmentPath | cut -d'=' -sf2)
readonly CASA_SERVICE_PATH

CASA_CONF=$( grep -i ExecStart= "${CASA_SERVICE_PATH}" | cut -d'=' -sf2 | cut -d' ' -sf3)
if [[ -z "${CASA_CONF}" ]]; then
    CASA_CONF=/etc/casaos/user-service.conf
fi

CASA_DB_PATH=$( (grep -i dbpath "${CASA_CONF}" || echo "/var/lib/casaos/db") | cut -d'=' -sf2 | xargs )
readonly CASA_DB_PATH

CASA_DB_FILE=${CASA_DB_PATH}/user-service.db

readonly aCOLOUR=(
    '\e[38;5;154m' # green  	| Lines, bullets and separators
    '\e[1m'        # Bold white	| Main descriptions
    '\e[90m'       # Grey		| Credits
    '\e[91m'       # Red		| Update notifications Alert
    '\e[33m'       # Yellow		| Emphasis
)

Show() {
    # OK
    if (($1 == 0)); then
        echo -e "${aCOLOUR[2]}[$COLOUR_RESET${aCOLOUR[0]}  OK  $COLOUR_RESET${aCOLOUR[2]}]$COLOUR_RESET $2"
    # FAILED
    elif (($1 == 1)); then
        echo -e "${aCOLOUR[2]}[$COLOUR_RESET${aCOLOUR[3]}FAILED$COLOUR_RESET${aCOLOUR[2]}]$COLOUR_RESET $2"
    # INFO
    elif (($1 == 2)); then
        echo -e "${aCOLOUR[2]}[$COLOUR_RESET${aCOLOUR[0]} INFO $COLOUR_RESET${aCOLOUR[2]}]$COLOUR_RESET $2"
    # NOTICE
    elif (($1 == 3)); then
        echo -e "${aCOLOUR[2]}[$COLOUR_RESET${aCOLOUR[4]}NOTICE$COLOUR_RESET${aCOLOUR[2]}]$COLOUR_RESET $2"
    fi
}

Warn() {
    echo -e "${aCOLOUR[3]}$1$COLOUR_RESET"
}

trap 'onCtrlC' INT
onCtrlC() {
    echo -e "${COLOUR_RESET}"
    exit 1
}

if [[ ! -x "$(command -v ${CASA_EXEC})" ]]; then
    Show 2 "${CASA_EXEC} is not detected, exit the script."
    exit 1
fi

while true; do
    echo -n -e "         ${aCOLOUR[4]}Do you want delete user database? Y/n :${COLOUR_RESET}"
    read -r input
    case $input in
    [yY][eE][sS] | [yY])
        REMOVE_USER_DATABASE=true
        break
        ;;
    [nN][oO] | [nN])
        REMOVE_USER_DATABASE=false
        break
        ;;
    *)
        echo -e "         ${aCOLOUR[3]}Invalid input, please try again.${COLOUR_RESET}"
        ;;
    esac
done

while true; do
    echo -n -e "         ${aCOLOUR[4]}Do you want delete user directory? Y/n :${COLOUR_RESET}"
    read -r input
    case $input in
    [yY][eE][sS] | [yY])
        REMOVE_USER_DIRECTORY=true
        break
        ;;
    [nN][oO] | [nN])
        REMOVE_USER_DIRECTORY=false
        break
        ;;
    *)
        echo -e "         ${aCOLOUR[3]}Invalid input, please try again.${COLOUR_RESET}"
        ;;
    esac
done

Show 2 "Stopping ${CASA_SERVICE}..."
systemctl disable --now "${CASA_SERVICE}" || Show 3 "Failed to disable ${CASA_SERVICE}"

rm -rvf "$(which ${CASA_EXEC})" || Show 3 "Failed to remove ${CASA_EXEC}"
rm -rvf "${CASA_CONF}" || Show 3 "Failed to remove ${CASA_CONF}"

if [[ "${REMOVE_USER_DATABASE}" == true ]]; then
    rm -rvf "${CASA_DB_FILE}" || Show 3 "Failed to remove ${CASA_DB_FILE}"
fi

if [[ "${REMOVE_USER_DIRECTORY}" == true ]]; then
    Show 2 "Removing user directories..."
    rm -rvf /var/lib/casaos/[1-9]*
fi
