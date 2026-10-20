# Homebrew's postgresql@N formulas are keg-only and link only versioned
# binaries (psql-18, pg_dump-18...) into bin. Put the newest installed
# version first in PATH to get plain psql, pg_dump, etc.
if [[ -n "$BREW_PATH" ]]; then
    pg_latest=(${BREW_PATH%/bin}/opt/postgresql@<->(N-/nOn[1]))
    if [[ -n "$pg_latest" && ":$PATH:" != *":$pg_latest/bin:"* ]]; then
        export PATH="$pg_latest/bin:$PATH"
    fi
    unset pg_latest
fi
