thepiratedb
===========

Download your SQLite mirror of The Pirate Bay using multiple concurrent keep-alive HTTPS connections.

Installation
------------

`go get github.com/FiloSottile/thepiratedb`

Usage
-----

```
Usage: thepiratedb [options] runnersNum maxTries
  -log=false: read the numbers from stdin
  -start=0: the starting torrent number
```

* `runnersNum`: number of downloading goroutines (suggested 500 on a 1Gbit connection)
* `maxTries`: retry attempts before skipping a torrent (suggested 5)
* `log`: download the torrent numbers received on stdin, to go over failed torrents in the log
* `start`: the torrent number to start from, for resuming and updating

If `-start` and `-log` are not set the database will be truncated.

Env var `DEBUG` enables a verbose exit-on-error mode.

An example `-log` invocation:

```
egrep -v "(Processing torrent|sql UNIQUE constraint failed)" thepirate.log | egrep "^2014/04" | egrep -o "[0-9]{5,}" | thepiratedb -log 500 5
```

An example `-start` invocation:

```
thepiratedb -start=$(sqlite3 thepirate.db "SELECT MAX(Id) FROM Torrents;") 500 5
```

Example row
------

    sqlite> SELECT * FROM Torrents WHERE Title LIKE "Ubuntu%" ORDER BY Seeders DESC LIMIT 1;

<table style="white-space:nowrap;">

<TR><TH>Id</TH>
<TH>Title</TH>
<TH>Category</TH>
<TH>Size</TH>
<TH>Seeders</TH>
<TH>Leechers</TH>
<TH>&nbsp;Uploaded&nbsp;</TH>
<TH>Uploader</TH>
<TH>Files_num</TH>
<TH>Description</TH>
<TH>Magnet</TH>
</TR>
<TR><TD>9059456</TD>
<TD>Ubuntu&nbsp;13.10&nbsp;Desktop&nbsp;Live&nbsp;ISO&nbsp;amd64</TD>
<TD>Applications&nbsp;&gt;&nbsp;UNIX</TD>
<TD>925892608</TD>
<TD>94</TD>
<TD>7</TD>
<TD>2013-10-17 12:58:32</TD>
<TD>Plan9x128</TD>
<TD>1</TD>
<TD>Ubuntu&nbsp;13.10&nbsp;Desktop&nbsp;Live&nbsp;Image&nbsp;(amd64/64bit)</TD>
<TD>magnet:?xt=urn:btih:e3811b9539cacff680e418124272177c47477157&amp;dn=Ubuntu+13.10+Desktop+Live+ISO+amd64&amp;tr=udp%3A%2F%2Ftracker.openbittorrent.com%3A80&amp;tr=udp%3A%2F%2Ftracker.publicbt.com%3A80&amp;tr=udp%3A%2F%2Ftracker.istole.it%3A6969&amp;tr=udp%3A%2F%2Ftracker.ccc.de%3A80&amp;tr=udp%3A%2F%2Fopen.demonii.com%3A1337</TD>
</TR>

</table>

Schema
------

```
sqlite> .schema
CREATE TABLE "Torrents" (
"Id" INTEGER PRIMARY KEY,
"Title" TEXT,
"Category" TEXT,
"Size" INTEGER,
"Seeders" INTEGER,
"Leechers" INTEGER,
"Uploaded" TEXT,
"Uploader" TEXT,
"Files_num" INTEGER,
"Description" TEXT,
"Magnet" TEXT
);
CREATE INDEX "TITLE" ON "Torrents" ("Title");
```
