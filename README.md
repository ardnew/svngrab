# svngrab
Export and merge paths from SVN repositories

## Installation

Use the `go get` tool:

```
go get -v github.com/ardnew/svngrab
```

## Usage

`svngrab` performs all of its operations according to the contents of a configuration file in YAML format.

The following example configuration file demonstrates a lot of its behavior:

```yaml
export:
    RepositoryA:
        repo: https://host/svn/a
        path: trunk
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
            output: ./MyPackage-${DATETIME}.zip
            overwrite: true
            method: zip
            level: 9
```
