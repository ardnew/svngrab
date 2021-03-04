package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ardnew/svngrab/config"
	"github.com/ardnew/svngrab/log"
	"github.com/ardnew/svngrab/repo"
	"github.com/ardnew/svngrab/run"
)

func usage(set *flag.FlagSet, separated, detailed bool) {
	exe := filepath.Base(executablePath())
	if separated {
		fmt.Fprintln(os.Stderr, "--")
	}
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  "+exe, "[options]", "[VAR=VAL ...]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "options:")
	set.PrintDefaults()
	if detailed {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "variables:")
		fmt.Fprintln(os.Stderr, "  Several elements of the configuration file support builtin and user-defined")
		fmt.Fprintln(os.Stderr, "  variables. Variable definitions are provided as command-line arguments of the")
		fmt.Fprintln(os.Stderr, "  form VAR=VAL. There should be no quotes surrounding VAL; however, if VAL")
		fmt.Fprintln(os.Stderr, "  contains spaces or other special characters, the entire argument may be")
		fmt.Fprintln(os.Stderr, "  enclosed with quotes, such as \"VAR=V A L\".")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "  With the variable definition VAR=VAL, the variable may be referenced in the")
		fmt.Fprintln(os.Stderr, "  configuration file as $VAR. A simple single-pass string substitution is")
		fmt.Fprintln(os.Stderr, "  performed to replace all occurrences of $VAR with VAL.")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "  The following builtin variables are always available, but may be overridden")
		fmt.Fprintln(os.Stderr, "  with definitions provided as command-line arguments:")
		fmt.Fprintln(os.Stderr, "  	$DATE       # current local date (\"YYYYMMDD\")")
		fmt.Fprintln(os.Stderr, "  	$DATETIME   # current local date-time (\"YYYYMMDD-hhmmss\")")
		fmt.Fprintln(os.Stderr)
	}
}

func main() {

	var configFilePath string
	var helpFlag bool

	flag.StringVar(&configFilePath, "f", filepath.Base(defaultConfigFilePath()),
		"Use configuration file at `path`")
	flag.BoolVar(&helpFlag, "h", false,
		"show the extended help cruft")
	flag.Usage = func() { usage(flag.CommandLine, false, false) }
	flag.Parse()

	if helpFlag {
		usage(flag.CommandLine, false, true)
		os.Exit(0)
	}

	flags := flagsProvided(flag.CommandLine)

	_, configFileProvided := flags["f"]

	// the defaults will show only the file name of the config file path, but we
	// want to use the full path to working dir if the user didn't provide it.
	if !configFileProvided {
		configFilePath = defaultConfigFilePath()
	}

	if configFilePath == "" {
		fmt.Fprintln(os.Stderr, "error:", "no configuration file defined")
		usage(flag.CommandLine, true, false)
		os.Exit(1)
	}

	vars, _ := userVariables(flag.Args()...)

	switch err := run.Run(log.New(os.Stdout), configFilePath, vars).(type) {
	case config.DirectoryNotFoundError:
		os.Exit(10)
	case config.ConfigFileNotFoundError:
		if !configFileProvided {
			usage(flag.CommandLine, true, false)
		}
		os.Exit(11)
	case config.InvalidPathError:
		if !configFileProvided {
			usage(flag.CommandLine, true, false)
		}
		os.Exit(12)
	case config.NotRegularFileError:
		if !configFileProvided {
			usage(flag.CommandLine, true, false)
		}
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

func executablePath() string {
	exe, err := os.Executable()
	if nil != err {
		panic("error: cannot determine executable: " + err.Error())
	}
	return exe
}

func defaultConfigFilePath() string {
	dir, err := os.Getwd()
	if nil != err {
		panic("error: cannot determine working directory: " + err.Error())
	}
	name := filepath.Base(executablePath())
	if ext := filepath.Ext(name); "" != ext {
		name = strings.TrimSuffix(name, ext)
	}
	return filepath.Join(dir, name+".yml")
}

func flagsProvided(set *flag.FlagSet) map[string]flag.Value {
	m := map[string]flag.Value{}
	set.Visit(func(f *flag.Flag) { m[f.Name] = f.Value })
	return m
}

func userVariables(argv ...string) (vars map[string]string, args []string) {
	vars = map[string]string{}
	args = []string{}
	for _, a := range argv {
		if i := strings.IndexRune(a, '='); i > -1 {
			v := ""
			if len(a) > i {
				v = a[i+1:]
			}
			vars["$"+a[:i]] = v
		} else {
			args = append(args, a)
		}
	}
	return
}
