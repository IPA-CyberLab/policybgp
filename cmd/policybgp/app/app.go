package app

import (
	"context"
	"io"

	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/IPA-CyberLab/policybgp/cmd/policybgp/serve"
)

func beforeImpl(ctx context.Context, cmd *cli.Command) error {
	var logger *zap.Logger
	if loggeri, ok := cmd.Metadata["Logger"]; ok {
		logger = loggeri.(*zap.Logger)
	} else {
		cfg := zap.NewProductionConfig()
		if cmd.Bool("verbose") {
			cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		}
		cfg.DisableCaller = true
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		//cfg.Encoding = "console"

		var err error
		logger, err = cfg.Build(
			zap.AddStacktrace(zap.NewAtomicLevelAt(zap.DPanicLevel)))
		if err != nil {
			return err
		}
	}
	zap.ReplaceGlobals(logger)

	return nil
}

var Command = &cli.Command{
	Name:  "policybgp",
	Usage: "Inject policy routing config via BGP",

	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "verbose",
			Usage: "enable verbose logging",
		},
	},

	Commands: []*cli.Command{
		serve.Command,
	},
	Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
		if err := beforeImpl(ctx, cmd); err != nil {
			// Print error message to stderr
			cmd.Writer = cmd.ErrWriter

			// Suppress help message on app.Before() failure.
			cli.HelpPrinter = func(_ io.Writer, _ string, _ interface{}) {}
			return ctx, err
		}

		return ctx, nil
	},
	After: func(ctx context.Context, cmd *cli.Command) error {
		_ = zap.L().Sync()
		return nil
	},
}
