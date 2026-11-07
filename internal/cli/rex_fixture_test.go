package cli

import (
	"context"
	"os"
	"testing"

	"github.com/itsjustaplant/rexp/internal/hooks"
)

// Session fixtures expose server identity without adding it to their mutation
// logs. Observer fixtures acknowledge subscription then disconnect, so no test
// worker is left running; hooks tests cover actual close events separately.
const rexIdentityStub = `if [ "$1" = whoami ]; then
  printf '%s\n' '{"host_id":"test-host","account":"test-account"}'
  exit 0
fi
if [ "$1" = do ] && [ "$2" = --all ]; then
  printf 'rexp-hook-ready\n'
  exit 0
fi
`

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == hooks.Mode {
		if err := hooks.Worker(context.Background(), os.Args[2:], os.Stdout, os.Stderr); err != nil {
			os.Stderr.WriteString(err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}
