#!/usr/bin/env bash
set -euo pipefail

# Start the official MySQL entrypoint in background.
/usr/local/bin/docker-entrypoint.sh mysqld &
pid=$!

# Wait for MySQL to be ready.
export MYSQL_PWD="${MYSQL_ROOT_PASSWORD}"
ready=0
for _ in $(seq 1 120); do
  if ! kill -0 "$pid" 2>/dev/null; then
    echo "mysqld exited before ready." >&2
    wait "$pid"
    exit 1
  fi
  if mysqladmin ping -h 127.0.0.1 -uroot --silent; then
    ready=1
    break
  fi
  sleep 1
done

if [ "$ready" -ne 1 ]; then
  echo "MySQL not ready, seed skipped." >&2
  wait "$pid"
  exit 1
fi

# Run all SQL under /seed-sql on every start. Let SQL control idempotency.
SEED_DIR=/seed-sql

if [ -d "${SEED_DIR}" ]; then
  while IFS= read -r -d '' file; do
    case "${file}" in
      */bootstrap/*)
        mysql -h 127.0.0.1 -uroot < "${file}"
        ;;
      *)
        mysql -h 127.0.0.1 -uroot --database=ran-feed < "${file}"
        ;;
    esac
  done < <(find "${SEED_DIR}" -type f -name '*.sql' -print0 | sort -z)
fi

wait "$pid"
