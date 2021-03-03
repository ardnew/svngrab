package config

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Type definitions for various errors raised by config package.
type (
	DirectoryNotFoundError  string
	ConfigFileNotFoundError string
	InvalidPathError        string
	NotRegularFileError     string
	FileExistsError         string
)

// Error returns the error message for DirectoryNotFoundError.
func (e DirectoryNotFoundError) Error() string {
	return "directory not found: " + string(e)
}

// Error returns the error message for ConfigFileNotFoundError.
func (e ConfigFileNotFoundError) Error() string {
	return "configuration file not found: " + string(e)
}

// Error returns the error message for InvalidPathError.
func (e InvalidPathError) Error() string {
	return "invalid file path: " + string(e)
}

// Error returns the error message for NotRegularFileError.
func (e NotRegularFileError) Error() string {
	return "not a regular file: " + string(e)
}

// Error returns the error message for FileExistsError.
func (e FileExistsError) Error() string {
	return "file already exists: " + string(e)
}

// Config represents a configuration file, containing the repositories to
// export and how to package them.
type Config struct {
	path    string
	Export  ExportMap  `yaml:"export"`
	Package PackageMap `yaml:"package"`
}

// ExportMap represents named SVN repository paths to export.
// The keys of this map may be used as reference by other operations in the
// configuration file.
type ExportMap map[string]ExportConfig

// ExportConfig represents the configuration for a single repository.
type ExportConfig struct {
	Repo  string `yaml:"repo"`
	Path  string `yaml:"path"`
	Local string `yaml:"local"`
	Last  string `yaml:"last"`
}

// urlProtocol is a regular expression that matches protocol string prefixes of
// URLs, up to and including the leading slashes.
// TODO: is this correct enough? Are there false-positives?
var urlProtocol = regexp.MustCompile(`^\s*[a-zA-Z]+://`)

// Url returns the remote URL of the SVN repository.
func (e *ExportConfig) Url() string {
	// remove the protocol prefix if it exists, because Join calls Clean, which
	// replaces double separators with a single separator, for example:
	//   "https://github.com" -> "http:/github.com"
	if i := urlProtocol.FindStringIndex(e.Repo); nil != i {
		return e.Repo[i[0]:i[1]] + path.Join(e.Repo[i[1]:], e.Path)
	}
	return path.Join(e.Repo, e.Path)
}

// Wc returns the local working path of the exported SVN repository.
func (e *ExportConfig) Wc() string {
	return filepath.Join(e.Local, e.Path)
}

// LastValid returns true if and only if Last is a valid SVN revision
// identifier.
func (e *ExportConfig) LastValid() bool {
	// TODO: figure out valid rules for a peg or revision identifier
	return e.Last != ""
}

// PackageMap represents all package operations to perform.
type PackageMap map[string]PackageConfig

// PackageConfig represents the configuration for a single package destination.
type PackageConfig struct {
	Roster   bool           `yaml:"roster"`
	Include  IncludeList    `yaml:"include"`
	Compress CompressConfig `yaml:"compress"`
}

// IncludeList represents the list of repositories to include in a package.
type IncludeList []IncludeMap

// IncludeMap associates a single named repository to a list of mapping
// configurations.
type IncludeMap map[string]IncludePathList

// IncludePathList contains a list of mapping configurations for a single
// repository.
type IncludePathList []IncludePathConfig

// IncludePathConfig represents a mapping configuration for a single path in a
// repository to its destination path in a package.
type IncludePathConfig struct {
	Repo     string   `yaml:"repo"`
	Package  string   `yaml:"package"`
	Conflict string   `yaml:"conflict"`
	Symlinks string   `yaml:"symlinks"`
	Ignore   []string `yaml:"ignore"`
}

// CompressConfig represents the configuration for a single compressed archive.
type CompressConfig struct {
	Output    string `yaml:"output"`
	Overwrite bool   `yaml:"overwrite"`
	Method    string `yaml:"method"`
	Level     int    `yaml:"level"`
}

// Parse parses the configuration file into the returned Config struct.
// Returns a nil Config and descriptive error if the given path is invalid or
// the configuration file could not be parsed.
func Parse(filePath string) (*Config, error) {

	dir := filepath.Dir(filePath)
	dstat, derr := os.Stat(dir)
	if os.IsNotExist(derr) {
		return nil, DirectoryNotFoundError(dir)
	} else if !dstat.IsDir() {
		return nil, InvalidPathError(dir)
	}

	fstat, ferr := os.Stat(filePath)
	if os.IsNotExist(ferr) {
		return nil, ConfigFileNotFoundError(filePath)
	} else if uint32(fstat.Mode()&os.ModeType) != 0 {
		return nil, NotRegularFileError(filePath)
	}

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	cfg := &Config{path: filePath}

	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// Write formats and writes the receiver configuration to disk.
// Returns an error if formatting or writing fails.
func (cfg *Config) Write() error {
	data, err := yaml.Marshal(cfg)
	if nil != err {
		return err
	}
	info, err := os.Stat(cfg.path)
	if nil != err {
		return err
	}
	return ioutil.WriteFile(cfg.path, data, info.Mode().Perm())
}
