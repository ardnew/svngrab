package run

import (
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ardnew/svngrab/config"
	"github.com/ardnew/svngrab/log"
	"github.com/ardnew/svngrab/repo"

	"github.com/mholt/archiver/v3"
	"github.com/otiai10/copy"
)

// Type definitions for various errors raised by run package.
type (
	InvalidIgnorePattern  string
	InvalidCompressMethod string
)

// Error returns the string representation of InvalidIgnorePattern
func (e InvalidIgnorePattern) Error() string {
	return "invalid ignore pattern: " + string(e)
}

// Error returns the string representation of InvalidCompressMethod
func (e InvalidCompressMethod) Error() string {
	return "invalid compress method: " + string(e)
}

// Constants defining default behaviors for file copy operations.
const (
	DefaultSymlinkAction   = copy.Skip
	DefaultDirExistsAction = copy.Merge
)

var Variable = map[string]string{
	"$DATE":     time.Now().Local().Format("20060102"),
	"$DATETIME": time.Now().Local().Format("20060102-150405"),
}

// Run executes the main program logic using the given log and configuration
// file path.
func Run(l *log.Log, path string, vars map[string]string) error {

	// copy the user variables definitions into our variable map.
	for ident, value := range vars {
		Variable[ident] = value
	}

	// parse the configuration file if it is valid YAML format.
	l.Infof("conf", "parsing configuration file: %s ...", path)
	cfg, err := config.Parse(path)
	l.Eolf("conf", err, " (ok)")
	if nil != err {
		return err
	}

	// create a mapping of export identifiers to actual VCS repository objects.
	reps := map[string]*repo.Repo{}

	// verify we can connect to each of the repository objects.
	for name, expo := range cfg.Export {

		// perform string replacement with variables on the name and export fields.
		for ident, value := range Variable {
			name = strings.ReplaceAll(name, ident, value)
			expo.Repo = strings.ReplaceAll(expo.Repo, ident, value)
			expo.Path = strings.ReplaceAll(expo.Path, ident, value)
			expo.Local = strings.ReplaceAll(expo.Local, ident, value)
		}

		l.Infof("repo", "initializing repostiory: %s ...", name)
		rep, err := repo.New(expo)
		l.Eolf("repo", err, " (ok)")
		if nil != err {
			return err
		}

		l.Infof("ping", "checking repository status: %s ...", name)
		_, err = rep.IsConnected()
		l.Eolf("ping", err, " (online)")
		if nil != err {
			return err
		}

		// install the repository reference in our map so that it can be referenced
		// in the package rules.
		reps[name] = rep
	}

	// export each of the repositories to a local working directory.
	for name, rep := range reps {
		var vers string
		mode, _ := rep.Exporter()
		l.Infof(mode.String(), "%s -> %s", rep.Remote(), rep.LocalPath())
		err := rep.Export()
		if nil == err {
			vers, err = rep.Revision()
		}
		l.Eolf(mode.String(), err, " (%s)", vers)
		if nil != err {
			return err
		}
		// update the last revision in the Config struct
		if expo, ok := cfg.Export[name]; ok {
			expo.Last = vers
			cfg.Export[name] = expo
		}
	}

	// parse the configuration file if it is valid YAML format.
	l.Infof("conf", "writing repository revisions: %s ...", path)
	err = cfg.Write()
	l.Eolf("conf", err, " (ok)")
	if nil != err {
		return err
	}

	// walk over each declared output package
	for pkgPath, pkg := range cfg.Package {

		// perform string replacement with variables on the package path.
		for ident, value := range Variable {
			pkgPath = strings.ReplaceAll(pkgPath, ident, value)
		}

		// walk over each repository we are copying content from for the current
		// output package.
		for _, inc := range pkg.Include {

			var srcPath string
			var incList config.IncludePathList

			for path, list := range inc { // only 1 key-value pair
				// perform string replacement with variables on the include path.
				for ident, value := range Variable {
					path = strings.ReplaceAll(path, ident, value)
				}
				srcPath = path
				incList = list
				if rep, isRepo := reps[path]; isRepo {
					srcPath = rep.LocalPath()
				}
			}

			// walk over each include operation for the current repository.
			for _, op := range incList {
				// check if there is a copy operation
				if cp := op.Copy; cp.Repo != "" && cp.Package != "" {
					// perform string replacement with variables on the copy fields.
					for ident, value := range Variable {
						cp.Repo = strings.ReplaceAll(cp.Repo, ident, value)
						cp.Package = strings.ReplaceAll(cp.Package, ident, value)
						for i := range cp.Ignore {
							cp.Ignore[i] = strings.ReplaceAll(cp.Ignore[i], ident, value)
						}
					}
					src, dst, opt, err := copyOptions(srcPath, pkgPath, cp)
					l.Infof("copy", "%s -> %s", src, dst)
					if nil == err {
						err = copy.Copy(src, dst, opt)
					}
					l.Eolf("copy", err, " (ok)")
					if nil != err {
						return err
					}
				}
			}
		}

		// create a compressed archive of the package if the output path is defined.
		if pkg.Compress.Output != "" {
			// perform string replacement with variables on the output path.
			for ident, value := range Variable {
				pkg.Compress.Output =
					strings.ReplaceAll(pkg.Compress.Output, ident, value)
			}
			arcPath, arc, err := makeArchiver(pkgPath, pkg.Compress)
			l.Infof("pack", "%s -> %s", pkgPath, arcPath)
			if nil == err {
				err = arc.Archive([]string{pkgPath}, arcPath)
			}
			l.Eolf("pack", err, " (ok)")
			if nil != err {
				return err
			}
		}
	}

	return nil
}

func copyOptions(srcPath, pkgPath string, cfg config.IncludeCopyConfig) (string, string, copy.Options, error) {
	// if repo path is not an asbolute path, append it to the repository local
	// working copy path.
	src := cfg.Repo
	if !filepath.IsAbs(src) {
		src = filepath.Join(srcPath, src)
	}
	// if destination path is not an absolute path, append it to the package root
	// path.
	dst := cfg.Package
	if !filepath.IsAbs(dst) {
		dst = filepath.Join(pkgPath, dst)
	}
	// convert the given copy option strings to their enumerated values.
	symlinks := symlinkAction(cfg.Symlinks)
	conflict := dirExistsAction(cfg.Conflict)
	skip, err := skipFunc(cfg.Ignore...)
	// construct a copy.Options struct with given configuration.
	return src, dst, copy.Options{
		OnSymlink:     func(s string) copy.SymlinkAction { return symlinks },
		OnDirExists:   func(s, d string) copy.DirExistsAction { return conflict },
		Skip:          func(s string) (bool, error) { return skip(s), nil },
		Sync:          true,
		PreserveTimes: true,
	}, err
}

func symlinkAction(action string) copy.SymlinkAction {
	switch strings.ToLower(action) {
	case "deep":
		return copy.Deep
	case "shallow":
		return copy.Shallow
	case "skip":
		return copy.Skip
	}
	return DefaultSymlinkAction
}

func dirExistsAction(action string) copy.DirExistsAction {
	switch strings.ToLower(action) {
	case "merge":
		return copy.Merge
	case "replace":
		return copy.Replace
	case "skip", "ignore", "untouchable":
		return copy.Untouchable
	}
	return DefaultDirExistsAction
}

func skipFunc(ignore ...string) (func(string) bool, error) {
	// convert the ignore strings to regexp patterns.
	ign := []*regexp.Regexp{}
	for _, s := range ignore {
		re, err := regexp.Compile(s)
		if nil != err {
			return nil, InvalidIgnorePattern(s)
		}
		ign = append(ign, re)
	}
	// return a function that checks if a given string matches any of the ignored
	// regexp patterns.
	return func(s string) bool {
		for _, re := range ign {
			if re.MatchString(s) {
				return true
			}
		}
		return false
	}, nil
}

func makeArchiver(pkgPath string, cfg config.CompressConfig) (string, archiver.Archiver, error) {

	var (
		arc archiver.Archiver
		ext string
		err error
	)

	// create an archiver for the declared compression method
	switch strings.ToLower(cfg.Method) {
	case "zip", ".zip":
		ext = ".zip"
		arc = &archiver.Zip{
			CompressionLevel:       cfg.Level,
			OverwriteExisting:      cfg.Overwrite,
			MkdirAll:               true,
			SelectiveCompression:   true,
			ImplicitTopLevelFolder: false,
			ContinueOnError:        false,
		}

	case "gz", ".gz", "tgz", ".tgz", "targz", "tar.gz", ".tar.gz":
		ext = ".tar.gz"
		arc = &archiver.TarGz{
			CompressionLevel: cfg.Level,
			Tar: &archiver.Tar{
				OverwriteExisting:      cfg.Overwrite,
				MkdirAll:               true,
				ImplicitTopLevelFolder: false,
				ContinueOnError:        false,
			},
		}

	case "bz2", ".bz2", "tbz", ".tbz", "tbz2", ".tbz2", "tarbz2", "tar.bz2", ".tar.bz2":
		ext = ".tar.bz2"
		arc = &archiver.TarBz2{
			CompressionLevel: cfg.Level,
			Tar: &archiver.Tar{
				OverwriteExisting:      cfg.Overwrite,
				MkdirAll:               true,
				ImplicitTopLevelFolder: false,
				ContinueOnError:        false,
			},
		}

	default:
		err = InvalidCompressMethod(cfg.Method)
	}

	if nil == err {
		if nil != arc.CheckExt(cfg.Output) {
			// remove existing extension if it exists, to replace with proper one
			if e := filepath.Ext(cfg.Output); "" != e {
				cfg.Output = strings.TrimSuffix(cfg.Output, e)
			}
			cfg.Output += ext
		}
	}

	return cfg.Output, arc, err
}
