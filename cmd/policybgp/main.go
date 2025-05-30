package main

import (
	"context"
	"errors"
	"os"

	"github.com/IPA-CyberLab/policybgp/cmd/policybgp/app"

	"go.uber.org/zap"
)

type ExitCoder interface {
	ExitCode() int
}

func ExitCodeOfError(err error) int {
	for {
		if ec, ok := err.(ExitCoder); ok {
			return ec.ExitCode()
		}

		if err = errors.Unwrap(err); err == nil {
			break
		}
	}

	return 1
}

func main() {
	if err := app.Command.Run(context.Background(), os.Args); err != nil {
		// omit stacktrace
		zap.L().WithOptions(zap.AddStacktrace(zap.FatalLevel)).Error(err.Error())
		os.Exit(ExitCodeOfError(err))
	}
}
