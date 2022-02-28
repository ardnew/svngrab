[docimg]:https://godoc.org/github.com/ardnew/svngrab?status.svg
[docurl]:https://godoc.org/github.com/ardnew/svngrab
[repimg]:https://goreportcard.com/badge/github.com/ardnew/svngrab
[repurl]:https://goreportcard.com/report/github.com/ardnew/svngrab

# svngrab
### Export and merge paths from SVN repositories

[![GoDoc][docimg]][docurl] [![Go Report Card][repimg]][repurl]

## Installation

#### Go modules: Go version 1.13 and later
Use the `go install` tool:

```
go install -v github.com/ardnew/svngrab@latest
```

#### Legacy: Go version 1.12 and earlier 
Use the `go get` tool:

```
GO111MODULE=off go get -v github.com/ardnew/svngrab
```

## Usage

`svngrab` performs all of its operations according to the contents of a configuration file in YAML format.

```
$ svngrab -h
usage:
  svngrab [options] [VAR=VAL ...]

options:
  -f path
        use configuration [f]ile at path (default "svngrab.yml")
  -h    show the extended [h]elp cruft
  -q    [q]uiet, output as little as possible
  -u    if all working copies are [u]p-to-date, exit immediately (code 2)
  -x path
        e[x]port results as shell environment script at path (or "-" stdout, "+" stderr)

variables:
  Several elements of the configuration file support builtin and user-defined
  variables. Variable definitions are provided as command-line arguments of the
  form VAR=VAL. There should be no quotes surrounding VAL; however, if VAL
  contains spaces or other special characters, the entire argument may be
  enclosed with quotes, such as "VAR=V A L".

  With the variable definition VAR=VAL, the variable may be referenced in the
  configuration file as $VAR. A simple single-pass string substitution is
  performed to replace all occurrences of $VAR with VAL.

  The following builtin variables are always available, but may be overridden
  with definitions provided as command-line arguments:
        $DATETIME   # current local date-time ("YYYYMMDD-hhmmss")
```

#### Configuration

The following example configuration file demonstrates a lot of its behavior:

```yaml
export:
    RepositoryA:
        repo: https://host/svn/a
        path: $BRANCH
        local: .svngrab/host/a/trunk
    RepositoryB:
        repo: https://host/svn/b
        path: branches/x
        local: .svngrab/host/b/branches/x
package:
    ./MyPackage/content:
        include:
            - RepositoryA:
                - copy: {repo: ./project/src, package: ./src, conflict: merge, symlinks: deep, ignore: [.svn, .o$, .a$]}
                - copy: {repo: ./project/include, package: ./include, conflict: replace, symlinks: shallow, ignore: [.svn]}
            - RepositoryB:
                - copy: {repo: ./Source, package: ./src, conflict: skip, symlinks: skip, ignore: [.svn]}
        compress:
            output: ./MyPackage-$DATETIME.zip
            overwrite: true
            method: zip
            level: 9
```
