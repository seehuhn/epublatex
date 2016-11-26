Epublatex
=========

A Go program to convert a subset of LaTeX into EPUB format.

Copyright (C) 2016  Jochen Voss

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Overview
--------

Epublatex can convert a subset of LaTeX into EPUB format.  For
compatibility with older ebook readers, formulas are rendered into
images (using pdflatex), and the rest of the LaTeX source is converted
into HTML; both the HTML and the rendered images are then combined
into an EPUB 3.0 file.

This program is not yet finished, and may not yet be usable for any
real projects.  Contributions in the form of patches would be very
welcome.

Experimenting with the Code
---------------------------

The project has the following dependencies:

  * go (to run/compile the source code)
  * pdflatex (to convert formulas to pdf)
  * ghostscript (to convert pdf to png)

To experiment with the current code::

  git checkout https://github.com/seehuhn/epublatex
  cd epublatex
  go run main.go examples/nonsense.tex

This will hopefully generate EPUB output in the file ``nonsense.epub``.
The code still has many rough edges and known problems:

  * Only a small subset of LaTeX is implemented.
  * I am still making large changes to the code without any attempt
    at backwards compatibility.

Note: The program keeps a cache of rendered images in some directory
(``$HOME/Library/Caches/de.seehuhn.ebook/maths/`` on MacOS, and
``$HOME/.cache/de.seehuhn.ebook/maths/`` on Linux).

Structure of the Code
---------------------

The epublatex source code is split over a number of packages:

  * ``github.com/seehuhn/epublatex/epub`` - Backend for storing HTML
    data in an EPUB 3.0 container.

  * ``github.com/seehuhn/epublatex/latex/scanner`` - Buffered reading of
    the source files, handling of include files.

  * ``github.com/seehuhn/epublatex/latex/tokenizer`` - Splits the
    LaTeX source into tokens.  This package has quite a bit of
    knowledge about LaTeX, in order to correctly identify macro
    arguments.

  * ``github.com/seehuhn/epublatex/latex/math`` - Uses pdflatex and
    ghostscript to convert mathematical formulas into PNG images.

  * ``github.com/seehuhn/epublatex/latex`` - Ties all the other
    components together, converts LaTeX to HTML, writes the result
    into an EPUB file.
