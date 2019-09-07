# ycopy

## Overview

This is an app for batch copying/downloading files. The list of files/urls is stored as a text file.

I forget my original use case for this, but once I added downloading of http(s) urls, I decided to put this in a git repo.

## Example Usage

Given a file `five-files.txt` that contains:

```text
# Note that empty lines are ignored, as are lines containing only whitespace.
# If the first character of a line is a '#', then the entire line is ignored.
# Here's the first local file that will be copied:
first.file

# Relative paths are allowed, and will be created in the target path if they don't exist:
foo\second.file
foo\bar\third.file

# Anything that begins with http:// or https:// is downloaded.
# In these cases, source path does nothing and is ignored.
http://url.example/fourth.file
http://url.example/path-in-url/fifth.file
```

Then here is an example run:

```text
$ pwd
/d
$ ycopy --list-file five-files.txt --source-path depot --target-path relative/path
2019/09/07 15:00:06 Starting 5 operations...
2019/09/07 15:00:07  1: D:\relative\path\first.file
2019/09/07 15:00:07  2: D:\relative\path\foo\second.file
2019/09/07 15:00:07  3: D:\relative\path\foo\bar\third.file
2019/09/07 15:00:07  4: D:\relative\path\fourth.file
2019/09/07 15:00:08  5: D:\relative\path\path-in-url\fifth.file
2019/09/07 15:00:08 Done.
```
