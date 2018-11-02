diplomagen is a program to replace texts and images in existing PDF files.

Building
--------
Make sure you have all dependencies:

```
go get -u gopkg.in/alecthomas/kingpin.v2
go get -u github.com/unidoc/unidoc/...
```

Then run:

```
go build -o diplomagen diplomagen.go
```


Usage
-----
To analyze a template PDF, run:

```
diplomagen strings 'FILE.pdf'
```

This lists all plaintext strings in the document, in the form: "{object id}:{line number}:{postscript string}".
These strings can be replaced with arbitrary new content by using the "replace line" patch in `patch` command. This patch takes the form "S{object id}:{line number}:{new postscript}". For a full example:

```
diplomagen patch --input 'test-samples/peter-wolf.pdf' 'S6:3826:[(random git user)]TJ' --output 'out.pdf'
```

Limitations
-----------

* Right now, it messes up the xref table, so this will need to be reconstructed using external tools.
* Some software prunes unused glyphs from embedded fonts, which will result in substitution characters or missing letters.

License
-------
Copyright © 2018 Managementboek.nl, all rights reserved.

Distribution of this utility and its source code is allowed under the terms of the GNU Affero General Public License version 3. See the following URL for details:
https://tldrlegal.com/license/gnu-affero-general-public-license-v3-(agpl-3.0)

This utility contains the following third-party libraries:

* [Kingpin](https://github.com/alecthomas/kingpin), available under the MIT license
* [Unidoc](https://github.com/unidoc/unidoc), available under the AGPL license
