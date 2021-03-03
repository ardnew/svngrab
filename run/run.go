package run

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ardnew/svngrab/config"
	"github.com/ardnew/svngrab/log"
	"github.com/ardnew/svngrab/repo"

	"github.com/otiai10/copy"
)

// Type definitions for various errors raised by run package.
type (
	InvalidIgnorePattern string
)

// Error returns the string representation of InvalidIgnorePattern
func (e InvalidIgnorePattern) Error() string {
	return "invalid ignore pattern: " + string(e)
}

// Constants defining default behaviors for file copy operations.
const (
	DefaultSymlinkAction   = copy.Skip
	DefaultDirExistsAction = copy.Merge
)

// Run executes the main program logic using the given log and configuration
// file path.
func Run(l *log.Log, path string) error {

	// parse the configuration file if it is valid YAML format.
	l.Infof("config", "parsing configuration file: %s ... ", path)
	cfg, err := config.Parse(path)
	l.Eolf("config", err, "ok")
	if nil != err {
		return err
	}

	// create a mapping of export identifiers to actual VCS repository objects.
	reps := map[string]*repo.Repo{}

	// verify we can connect to each of the repository objects.
	for name, expo := range cfg.Export {

		l.Infof("repo", "initializing repostiory: %s ... ", name)
		rep, err := repo.New(expo)
		l.Eolf("repo", err, "ok")
		if nil != err {
			return err
		}

		l.Infof("connect", "checking repository status: %s ... ", name)
		_, err = rep.IsConnected()
		l.Eolf("connect", err, "online")
		if nil != err {
			return err
		}

		reps[name] = rep
	}

	// export each of the repositories to a local working directory.
	for _, rep := range reps {
		var vers string
		mode, _ := rep.Exporter()
		l.Infof(mode.String(), "%s -> %s ", rep.Remote(), rep.LocalPath())
		err := rep.Export()
		if nil == err {
			vers, err = rep.Revision()
		}
		l.Eolf(mode.String(), err, "(%s)", vers)
		if nil != err {
			return err
		}
	}

	// walk over each declared output package
	for pkgPath, pkg := range cfg.Package {
		// walk over each repository we are copying content from for the current
		// output package.
		for _, inc := range pkg.Include {

			var srcPath string
			var incList config.IncludePathList

			for path, list := range inc { // only 1 key-value pair
				srcPath = path
				incList = list
				if rep, isRepo := reps[path]; isRepo {
					srcPath = rep.LocalPath()
				}
			}

			// walk over each copy mapping for the current repository in the current
			// package.
			for _, item := range incList {
				src, dst, opt, err := copyOptions(srcPath, pkgPath, item)
				l.Infof("copy", "%s -> %s", src, dst)
				if nil == err {
					err = copy.Copy(src, dst, opt)
				}
				l.Eolf("copy", err, "")
				if nil != err {
					return err
				}
			}
		}
	}

	return nil
}

func copyOptions(srcPath, pkgPath string, cfg config.IncludePathConfig) (string, string, copy.Options, error) {
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
