ChessBuddy
==========

Play chess with [Go][1], HTML5, [WebSockets][2] and random strangers!

* Demo: <http://www.tux21b.org:8000/>

Hint: Open the page in two different tabs, if there aren't any other
visitors around.


Quick Start
-----------

ChessBuddy is compatible with Go 1. Use the [go tool][3] to install the latest
version of ChessBuddy with the following command:

    go get github.com/tux21b/ChessBuddy

This will install a new command `ChessBuddy` in your path (usually
`$GOPATH/bin/ChessBuddy`). Start the HTTP server with the default arguments:

    ChessBuddy -http=:8000 -time=5m

Visit <http://localhost:8000/>, wait for a friend and start playing a game of
chess.

You can use `go get -u github.com/tux21b/ChessBuddy` to update ChessBuddy.


Features
--------

 * web service connects all visitors in pairs and maintains the chess games
 * JavaScript client displays the chess board using the HTML 5 canvas API
 * Time control: 5 minutes (configurable) per side, sudden death
 * move history displays all moves using standard algebraic notation (SAN)


Missing Features
----------------

* en passant attacks and promotion
* invite a specific person by sharing a custom URL


License
-------

ChessBuddy is distributed under the Simplified BSD License:

> Copyright © 2012 Christoph Hack. All rights reserved.
>
> Redistribution and use in source and binary forms, with or without
> modification, are permitted provided that the following conditions are met:
>
>    1. Redistributions of source code must retain the above copyright notice,
>       this list of conditions and the following disclaimer.
>
>    2. Redistributions in binary form must reproduce the above copyright
>       notice, this list of conditions and the following disclaimer in the
>       documentation and/or other materials provided with the distribution.
>
> THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDER ``AS IS'' AND ANY EXPRESS
> OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES
> OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO
> EVENT SHALL <COPYRIGHT HOLDER> OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT,
> INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING,
> BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
> DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY
> OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING
> NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE,
> EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
>
> The views and conclusions contained in the software and documentation are
> those of the authors and should not be interpreted as representing official
> policies, either expressed or implied, of the copyright holder.


[1]: http://golang.org/
[2]: http://dev.w3.org/html5/websockets/
[3]: http://golang.org/cmd/go/
