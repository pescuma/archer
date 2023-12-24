package main

import (
	"github.com/pescuma/go-build"
)

func main() {
	cfg := build.NewBuilderConfig()
	cfg.Archs = []string{
		"darwin/amd64",
		"darwin/arm64",
		"linux/386",
		"linux/amd64",
		//"windows/386", go-sqlite does not compile
		"windows/amd64",
	}

	b, err := build.NewBuilder(cfg)
	if err != nil {
		panic(err)
	}

	b.Targets.Add("gall", []string{"license-check", "generate", "build", "test", "zip"}, nil)

	err = b.RunTarget("gall")
	if err != nil {
		panic(err)
	}
}
