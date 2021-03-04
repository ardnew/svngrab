[docimg]:https://godoc.org/github.com/ardnew/svngrab?status.svg
[docurl]:https://godoc.org/github.com/ardnew/svngrab
[repimg]:https://goreportcard.com/badge/github.com/ardnew/svngrab
[repurl]:https://goreportcard.com/report/github.com/ardnew/svngrab

# svngrab
### Export and merge paths from SVN repositories

[![GoDoc][docimg]][docurl] [![Go Report Card][repimg]][repurl]

## Installation

Use the `go get` tool:

```
go get -v github.com/ardnew/svngrab
```

## Usage

`svngrab` performs all of its operations according to the contents of a configuration file in YAML format.

```
$ svngrab -h
usage:
  svngrab [options] [VAR=VAL ...]

options:
  -f path
        Use configuration file at path (default "svngrab.yml")
  -h    show the extended help cruft

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
        $DATE       # current local date ("YYYYMMDD")
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
