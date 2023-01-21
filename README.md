```
Usage:

    bla <paths> \
        f=<file match pattern> \
        p=<path match pattern> \
        c=<content match pattern>

Where the match pattern is a set of literals separted by two dots "..",
like: ..foo..bar.. or foo..bar

For files and paths, the pattern matches whole file or path. For content, the
pattern matches any part of the content (.. are added implicitly at the
beginning and the end of the pattern.)

Path and filenames are case-insensitive and the patterns must be lower-case.
Content is case sensitive unless one passes -i flag.

  -i	case insensitive content matches (slower)
  -v	verbose debug mode
```
