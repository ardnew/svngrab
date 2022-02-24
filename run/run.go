package run

import (
	"io"
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
	WorkingCopiesUpToDate bool
)

// Error returns the string representation of InvalidIgnorePattern
func (e InvalidIgnorePattern) Error() string {
	return "invalid ignore pattern: " + string(e)
}

// Error returns the string representation of InvalidCompressMethod
func (e InvalidCompressMethod) Error() string {
	return "invalid compress method: " + string(e)
}

// Error returns the string representation of WorkingCopiesUpToDate
func (e WorkingCopiesUpToDate) Error() string {
	return "all working copies up-to-date"
}

// Constants defining default behaviors for file copy operations.
const (
	DefaultSymlinkAction   = copy.Skip
	DefaultDirExistsAction = copy.Merge
)

var Variable = map[string]string{
	//	"$DATE":     time.Now().Local().Format("20060102"),
	"$DATETIME": time.Now().Local().Format("20060102-150405"),
}

// Run executes the main program logic using the given log and configuration
// file path.
func Run(l *log.Log, path string, sh *ShellEnv, update bool, vars map[string]string) error {

	// store each of our key-value string pairs to be written into our shell
	// environment script.
	defer sh.Close()

	// copy the user variables definitions into our variable map.
	for ident, value := range vars {
		Variable[ident] = value
		sh.Append("input variables", "VAR_"+ident, value)
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

		sh.Append(name, "REPO_"+name+"_URL",
			strings.TrimRight(expo.Repo, "/")+"/"+strings.TrimLeft(expo.Path, "/"))
		sh.Append(name, "REPO_"+name+"_LOCAL", expo.Local)
		// placeholders so we have each repository's entire info grouped together.
		// the Append method will notice we have a duplicate key.
		sh.Append(name, "REPO_"+name+"_PREVREV", "")
		sh.Append(name, "REPO_"+name+"_CURRREV", "")

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

	didUpdate := false
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
			if expo.Last != vers {
				didUpdate = true
			}
			sh.Append(name, "REPO_"+name+"_PREVREV", expo.Last)
			sh.Append(name, "REPO_"+name+"_CURRREV", vers)
			expo.Last = vers
			cfg.Export[name] = expo
		}
	}

	l.Infof("envi", "generating shell environment: %s ...", sh.Name)
	_, err = sh.Commit()
	l.Eolf("envi", err, " (ok)")
	if err != nil {
		return err
	}

	// return early if user provided update flag -u and we did not update
	// any working copy.
	if upToDate := WorkingCopiesUpToDate(update && !didUpdate); upToDate {
		l.Errorf("conf", "%s", upToDate)
		l.Break()
		return upToDate
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

// ShellEnv implements io.WriteCloser and provides storage for the exported
// shell environment script.
// It also provides methods for formatting and writing the stored contents.
type ShellEnv struct {
	Name   string
	Writer io.Writer // must never be nil
	Closer io.Closer // possibly nil (e.g., w = io.Discard)

	section []struct {
		name string
		env  *shellEnvSection
	}
}

func NewShellEnv(name string, writer io.Writer, closer io.Closer) *ShellEnv {
	return &ShellEnv{
		Name:   name,
		Writer: writer,
		Closer: closer,
		section: []struct {
			name string
			env  *shellEnvSection
		}{},
	}
}

func (s *ShellEnv) Write(p []byte) (n int, err error) {
	return s.Writer.Write(p)
}

func (s *ShellEnv) Close() error {
	if s.Closer != nil {
		return s.Closer.Close()
	}
	return nil
}

// Note that the newline character sequence depends on compile-time target OS,
// which is "\r\n" for Windows, "\n" for everyone else.
func (s *ShellEnv) String() string {
	var sb strings.Builder
	for n, sect := range s.section {
		if n > 0 {
			sb.WriteString(log.Eol)
		}
		sb.WriteString("# " + log.Eol)
		sb.WriteString("# " + sect.name + log.Eol)
		sb.WriteString("# " + log.Eol)
		sb.WriteString(sect.env.String())
	}
	return sb.String()
}

func (s *ShellEnv) Commit() (n int, err error) {
	// use the Writer member instead of the receiver ShellEnv so that we may take
	// advantage if the member implements the optimized WriteString method
	// (because ShellEnv does not/cannot implement WriteString).
	return io.WriteString(s.Writer, s.String())
}

var (
	reUnderscores = regexp.MustCompile("_+")
	reNonidents   = regexp.MustCompile("(^[^A-Z_]|[^A-Z0-9_]+)")
	//reUnescaped  = regexp.MustCompile("(^|[^\\])([\"`$])")
)

func (s *ShellEnv) Append(section, key, val string) {

	var env *shellEnvSection
	for _, sect := range s.section {
		if sect.name == section {
			env = sect.env
			break
		}
	}
	if env == nil {
		env = &shellEnvSection{}
		s.section = append(s.section,
			struct {
				name string
				env  *shellEnvSection
			}{
				name: section,
				env:  env,
			})
	}

	// Sanitize key for sh-compatible identifiers
	key = strings.ToUpper(strings.TrimSpace(key))
	key = reNonidents.ReplaceAllLiteralString(key, "_")
	key = reUnderscores.ReplaceAllLiteralString(key, "_")
	key = strings.Trim(key, "_")

	// Sanitize val for being enquoted with double-quotes ("") by inserting
	// an escape "\" before any symbol that delimits string interpolation.
	// Note that if the symbol has ANY number of preceding escapes "\", then it
	// will NOT have an escape inserted! This is a convoluted bug I don't
	// want to deal with at the moment, as the current behavior seems to
	// have the least surprising results.
	//val = reUnescaped.ReplaceAllString(val, `${1}\${2}`)

	// check if the given key already exists
	n := env.Len()
	for i := 0; i < n; i++ {
		if env.key[i] == key {
			env.val[i] = val // found key, update existing value
			return           // do not add new elements
		}
	}

	// add key-value pair to end of section
	env.key = append(env.key, key)
	env.val = append(env.val, val)
	env.count++
}

type shellEnvSection struct {
	count int
	key   []string
	val   []string
}

func (s *shellEnvSection) Len() int {
	n := s.count
	if nk := len(s.key); nk < n {
		n = nk
	}
	if nv := len(s.val); nv < n {
		n = nv
	}
	return n
}

// String creates a newline-delimited string, with each line containing the
// elements at that line's index from both key and val, separated by a single
// equals sign, and with val surrounded by double-quotes. For example:
//   key[0]="val[0]"
//   key[1]="val[1]"
// Note that the newline character sequence depends on compile-time target OS,
// which is "\r\n" for Windows, "\n" for everyone else.
func (s *shellEnvSection) String() string {
	var sb strings.Builder
	for i, n := 0, s.Len(); i < n; i++ {
		sb.WriteString(s.key[i] + `="` + s.val[i] + `"` + log.Eol)
	}
	return sb.String()
}
