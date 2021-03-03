package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ardnew/svngrab/config"
	"github.com/ardnew/svngrab/log"
	"github.com/ardnew/svngrab/repo"
	"github.com/ardnew/svngrab/run"
)

func usage(flags *flag.FlagSet) {
	exe, err := os.Executable()
	if nil != err {
		fmt.Fprintln(os.Stderr, "error: cannot determine executable:", err.Error())
	}
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  "+filepath.Base(exe), "[options]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "options:")
	flags.PrintDefaults()
}

func main() {

	var configFilePath string

	flag.StringVar(&configFilePath, "f", "", "Use configuration file at `path`")
	flag.Usage = func() { usage(flag.CommandLine) }
	flag.Parse()

	if configFilePath == "" {
		fmt.Fprintln(os.Stderr, "error:", "no configuration file defined")
		fmt.Fprintln(os.Stderr, "--")
		flag.Usage()
		os.Exit(0)
	}

	switch err := run.Run(log.New(os.Stdout), configFilePath).(type) {
	case config.DirectoryNotFoundError:
		os.Exit(10)
	case config.ConfigFileNotFoundError:
		os.Exit(11)
	case config.InvalidPathError:
		os.Exit(12)
	case config.NotRegularFileError:
		os.Exit(13)
	case config.FileExistsError:
		os.Exit(14)
	case repo.InvalidRepositoryError:
		os.Exit(20)
	case repo.ConnectionFailedError:
		os.Exit(21)
	case repo.ExportFailedError:
		os.Exit(22)
	case repo.UnknownRevisionError:
		os.Exit(23)
	case run.InvalidIgnorePattern:
		os.Exit(100)
	default:
		if nil != err {
			os.Exit(99)
		}
	}
}
