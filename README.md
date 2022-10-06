go-proj
=======

Description
-----------

PROJ wrapper for the transformation of points between different coorinate reference systems.

Changes
-------
v2 version of this package requires Proj > 4. The API of this package changed, please check the documentation.
You need to set the environment `PROJ_USE_PROJ4_INIT_RULES=YES` if you want to use Proj.4 init strings (`+init=epsg:xxx`) with backwards compatible axis orientation.

Installation
------------

This package can be installed with the go get command:

    go get github.com/omniscale/go-proj/v2

This package requires [proj](https://proj.org/) (`libproj-dev` on Ubuntu/Debian, `proj` in Homebrew).

Documentation
-------------

API documentation can be found here: http://godoc.org/github.com/omniscale/go-proj/v2


License
-------

MIT, see LICENSE file.

Author
------

Oliver Tonnhofer
