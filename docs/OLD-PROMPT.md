# About

`pi` is a package installer with following features

* Universal: `pi` is for installing packages from every ecosystem, python, java, javascript
  ec. It'll install the required package manager and then install the packages. It can also
  compile packages from the source for various languages.
  It can also use system installed packages.

* Sandboxing : Every external program is executed in a cave. There are going to be several
  sandboxing implementations, but currently  only bubblewrap on linux is supported,

* Workspace: `pi` treats a directory as workspace, and maintains `.local` folder tree by
  keeping symlinks of installed packages.The workspace may have multiple projects
  in multiple language, but use the same version of the installable depdendencies.
  `pi` will only update files in `~/.cache/pi/` and `{workspace}/.local`



# Design

## Invocation

`pi` is a non-interactive command line application, which can read and update the
file `pi.json` in the workspcae directory. An invocation of `pi` either results
in error message, report, or execution of a command in a sandbox, with environment
variables setup correctly to use the packages.

## Working

`pi` uses workspace to know the packages to be installed, installs the packages,
creates symlinks to the workspace HOME/.local/, setups the environment variables
and executes bubblewrap with the command ( or bash ).
To install the packages, it refers to recipes.

## Recipies
`pi` will read recipes to know how to install a package. These recipes are in
starlark language, and `pi` supplies several functions that can be used.

The recipe files are declarative, and provide pure functions to transform data
to make it usable by `pi`. As an example, the recipe for nodejs will have the
url to find json of list of distributions, a function to return versions from
the json ( which will use json parser supplied by `pi`), etc. The recipe will
not be able to an network or file system IO.  One recipe file can declare several
packages, or package ecosystem.

## Repositories
Recipe repository can contains several recipies. `pi` will download and index
recipes for quick lookup.
Repository can be distributed as git repo, archive files, or single file.
