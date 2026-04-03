package main

import (
	"github.com/adrianozp/gaardrail/cmd/api/options"
	"go.uber.org/fx"
)

func main() {
	fx.New(options.Options()).Run()
}
