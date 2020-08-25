#!/bin/sh
set -eo pipefail

log() {
	local type="$1"; shift
	printf '%s [Entrypoint] [%s] : %s\n' "$(date +'%Y/%m/%d %T')" "$type" "$*"
}

log_info() {
	log INFO "$@"
}

log_warn() {
	log WARN "$@" >&2
}

log_error() {
	log ERROR "$@" >&2
	exit 1
}

_main() {
    log_info "Processing input parameters..."
	# if command starts with an option, prepend prometheus-sql
	if [ "${1:0:1}" = '-' ]; then
		set -- "prometheus-sql" "$@"
	fi

    if [ "$1" = 'prometheus-sql' ]; then
        set -- "$@" "-host" "${PROMSQL_BIND_ADDRESS}" "-port" "${PROMSQL_PORT}" 
    fi

	exec "$@"
}

_main "$@"
