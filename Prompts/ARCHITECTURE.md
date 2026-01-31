# Modules


## Multi - processing

goroutine and 1 deep channels are used as far as possible.
Only in rare case, mutex may be employed to update shared data structures.

There can be several package, and `pi` will need to do several operations,
like downloading, extracting, compiling, symlinking etc.

The advantage of go is that we don't need to breakup these in multiple functions.
If a pkg has dependency on 5 packages, than it should be possible to process
these 5 package in parallel ( assuming they dont' have dependency on each other)
This can be done either by making a graph upfront, or letting graph be built
dynamically. `pi` allows this to be done dynamically.

At a very high level, the function `Install(packageDefinition) packageInstallation,error`
is async, reentrant, and waitable, and cachable, and serialized for same package.

## Recipe

** For first version, we are not going to have starlark based recipe.
We are going to have go functions providing packageDefinition implementation. **


## Caching

`pi` caches file system output of everything to avoid redoing ( downloading, extracting etc )


## Cave

## Utility modules

### display

Single point for any screen/file display and logging.
This can have a curses based imlementation.
Allows caller to create a 'task', and then submit the log/display/progress of it.
Flags determine what content goes to screen, logs etc.
Keeps track ot active tasks, and displays progress bars in text.

### archive

Has the code to unarchive all sort of archives.


### cave-bubblewrap

bubblewrap impementation of cave.


## Tooling
