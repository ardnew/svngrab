package repo

import (
	"github.com/ardnew/svngrab/config"

	"github.com/Masterminds/vcs"
)

// Type definitions for various errors raised by repo package.
type (
	InvalidRepositoryError string
	ConnectionFailedError  string
	ExportFailedError      string
	UnknownRevisionError   string
)

// Error returns the string representation of InvalidRepositoryError
func (e InvalidRepositoryError) Error() string {
	return "invalid repository: " + string(e)
}

// Error returns the string representation of ConnectionFailedError
func (e ConnectionFailedError) Error() string {
	return "failed to connect to repository: " + string(e)
}

// Error returns the string representation of ExportFailedError
func (e ExportFailedError) Error() string {
	return "failed to export repository: " + string(e)
}

// Error returns the string representation of UnknownRevisionError
func (e UnknownRevisionError) Error() string {
	return "cannot determine revision of repository: " + string(e)
}

// Repo contains a VCS repository object (SVN-only) combined with its options
// parsed from the configuration file.
type Repo struct {
	*vcs.SvnRepo
	cfg config.ExportConfig
}

// New returns a pointer to a new Repo object using the given configuration.
// A nil Repo pointer and non-nil error is returned if the VCS object could not
// be created from the configuration options.
func New(cfg config.ExportConfig) (*Repo, error) {
	svn, err := vcs.NewSvnRepo(cfg.Url(), cfg.Wc())
	if nil != err {
		return nil, InvalidRepositoryError(err.Error())
	}
	return &Repo{
		SvnRepo: svn,
		cfg:     cfg,
	}, nil
}

// Connect verifies communication with the remote repository, or returns an
// error if the connection fails.
func (r *Repo) IsConnected() (bool, error) {
	if !r.Ping() {
		return false, ConnectionFailedError(r.Remote())
	}
	return true, nil
}

// Exporter returns the VCS method (and its corresponding ExportMode) required
// to retrieve the remote repository.
// If a local working copy exists, the method returned is equivalent to an
// update; otherwise, working copy does not exist, the method is a checkout.
func (r *Repo) Exporter() (ExportMode, func() error) {
	if r.CheckLocal() {
		return UpdateMode, r.Update
	}
	return CheckoutMode, r.Get
}

// Export retrieves the remote repository by either update or checkout,
// depending on if the local working copy exists or not.
func (r *Repo) Export() error {
	_, fetch := r.Exporter()
	if err := fetch(); nil != err {
		return ExportFailedError(err.Error())
	}
	return nil
}

// Revision returns the repository revision of the local working copy.
func (r *Repo) Revision() (string, error) {
	vers, err := r.Version()
	if nil != err {
		return "", UnknownRevisionError(err.Error())
	}
	return vers, nil
}
