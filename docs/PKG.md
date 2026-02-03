# Package

Package can be of multiple type. Taking example of python, it can be
* python it self
* pip or pipx
* a python package installed by pip

A full pkg name is

    ecosystem:pkgname=version

ecosystem are the website which holds the package, like pip, npm, github, etc.
pkgname can have '/' or '@' etc, as it's allowed by several ecosystems
version can be a keyword if strats with a-z, or version if starts from digit.

keywords can be 'stable', 'lts', 'rc' etc.

* python <- latest stable version of python
* python=stable <- as above.
* python=3.12 <- latest stable version of 3.12 series
* python=3.12.1 <- exact version.

* pip:numpy
* npm:@gemini-cli/latest
