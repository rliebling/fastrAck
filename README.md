fastrAck
========
fastrAck is a faster ack (http://betterthangrep.com).  It allows you to search files in a directory tree based on regular expressions _very_ quickly.  It does this by building and maintaining an index.  fastrAck was built by stitching together software written by others.  I may have tweaked things a bit here and there, but all the heavy lifting was done by others.

## Performance
On a two year old windows laptop I can search a rails codebase of over 10,000 files in less than half a second. The initial index building takes < 10seconds.  Incremental updates to the index take less time.

## Installation

 * Have go installed (http://golang.org).
 * create a workspace directory for this project (eg `mkdir ~/fastrAck`)
 * set your GO environment variables GOROOT (to where go is installed) and GOPATH (to the workspace directory you just created, eg ~/fastrAck)
 * change dir to the workspace dir
 * execute this command `go get github.com/rliebling/fastrAck`
 * find your fastrAck executable in GOPATH/bin

## Usage
### Watch a directory tree
Build the index (stored in .cindex) from scratch and updates whenever a file changes (after a delay of 10 seconds).

`fastrAck -watch`

### Build the index once and exit
`fastrAck -index`

### Search
##### Search for the regex /foo.*bar/ across all files

`fastrAck foo.*bar`

##### Search for the regex /foo.*bar/ across all files whose paths match the regex /app/

`fastrAck -f app foo.*bar`


##### Search for the regex /foo.*bar/ across all files whose paths match the regex /vendor/ and but don't match the regexp /test/

`fastrAck -f vendor -F test foo.*bar`


## Status

Works on Windows, Linux and OSX.  

NOTE: on OSX, watching a directory maintains open file descriptors for (virtually) every file, which can run into the limit on OSX of around 10,000.  I'm hoping to resolve this, but for now the only solutions I know of are:

  1.  increase that limit (google for how to do that), or
  2.  don't watch the directory tree.  Just use the `-index` flag to build the index on demand.
  3.   use some other tool (eg http://www.rubyinside.com/watchr-more-than-an-automated-test-runner-4590.html) to run the index command whenever a file changes.

On Windows, as a quick way to get the ANSI color code support to work, I pipe the output through a ruby process with the win32console gem.  The code assumes ruby is in the path with this gem available (eg the command `ruby -rwin32console` should work.  If you run without color support (eg where the output is redirected or with --color=false) then you don't need ruby.

## Credits

The really heavy lifting here was done by Russ Cox in the project
http://code.google.com/p/codesearch. I modified the code a bit to make it clean up after itself better based on some patches that were posted for it.  

Handling of ANSI color codes for terminal output utilizes the terminal library from https://github.com/kless/terminal which I modified slightly to help it build on Windows.

Notifications of file changes (to trigger updating the index) is handled by Chris Howey's library https://github.com/howeyc/fsnotify

## License

