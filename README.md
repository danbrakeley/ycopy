# ycopy

## Overview

This is an app for batch copying/downloading files that are specified in a newline-separated text file.

I forget my original use case for this, but once I added downloading from web urls, I decided to put this in a git repo and add some more features.

## Example Usage

Given a file `five-files.txt` that contains:

```text
# Note that empty lines are ignored, as are lines containing only whitespace.
# If the first character of a line is a '#', then the entire line is ignored.
# Here's the first local file that will be copied:
first.file

# Relative paths are allowed, and will be created in the destination path if they don't exist:
foo\second.file
foo\bar\third.file

# Anything that begins with http:// or https:// is downloaded.
# For remote files like these, the source path supplied on the command line unused.
http://url.example/fourth.file
http://url.example/path-in-url/fifth.file
```

Then here is what an example run might look like:

```text
$ pwd
/d
$ ycopy --src depot --dest relative/path five-files.txt
2019/09/07 15:00:06 Starting 5 operations...
2019/09/07 15:00:07  1: D:\relative\path\first.file
2019/09/07 15:00:07  2: D:\relative\path\foo\second.file
2019/09/07 15:00:07  3: D:\relative\path\foo\bar\third.file
2019/09/07 15:00:07  4: D:\relative\path\fourth.file
2019/09/07 15:00:08  5: D:\relative\path\path-in-url\fifth.file
2019/09/07 15:00:08 Done.
```

## Todo

- ✓ ~~Specify threads on command line?~~
  - ✓ ~~re-write to put copy operations in go funcs~~
- ✓ ~~cli error display cleanup~~
  - ✓ ~~remove all commands (help)~~
- handle signals
  - ✓ ~~on ctrl-c, stop feeding workers and wait for running actions to complete~~
  - on second ctrl-c, abort transfers (delete partial files?)
- ✓ ~~properly detect http errors~~
- ✓ ~~logger~~
  - ✓ ~~supports ansi (when terminal connected)~~
  - ✓ ~~supports fixed lines (for progress bars)~~
- retries
- print error report at end (even if ctrl-c)
  - do not include errors that resulted in a success after retrying
  - just list one failed file per line
- scrape given url to generate single file list
- progress bars
  - per thread
  - overall/status
  - disable w/ --no-progress (allow colors, but just no fixed log lines)
- allow flags to be set after arguments?
- interactive
  - add thread
  - pause
- skip if destination file already exists
  - for local copies, allow time/size/other checks as well?
- performance - anecdotal evidence says it is plenty fast, but what about slower media?
  - large files?
  - memory usage?
