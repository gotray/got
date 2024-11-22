package got

import (
	"fmt"
	"os"

	"github.com/gotray/got/internal/env"
)

var (
	ProjectRoot string
)

func SetEnv() {
	if ProjectRoot == "" {
		fmt.Fprintf(os.Stderr, "WARNING: github.com/gotray/got.ProjectRoot is not set, compile with got to set it\n")
		return
	}
	envs, err := env.ReadEnv(ProjectRoot)
	if err != nil {
		panic(err)
	}
	for key, value := range envs {
		os.Setenv(key, value)
	}
}
