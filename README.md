```
Yet another file search tool. An equivalent of "find ... | egrep ...". The patterns are defined as lower-case literals and two dots "..", like:

    ..foo..    is  /.*foo.*/
    bar..      is /bar.*/
    ..foo..bar is /.*foo.*bar/

  -c string
    	Path to toml config file. If empty, default locations are checked.
  -f value
    	File filters.
  -nf value
    	File negative filters.
  -np value
    	Path negative filters.
  -p value
    	Path filters.
  -v	Verbose debug mode.
```
