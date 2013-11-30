/**
 * ChessBuddy - Play chess with Go, HTML5, WebSockets and random strangers
 *
 * Copyright (c) 2012 by Christoph Hack <christoph@tux21b.org>
 * All rights reserved. Distributed under the Simplified BSD License.
 */

var P = 1;
var N = 2;
var B = 3;
var R = 4;
var Q = 5;
var K = 6;

var WHITE = 8;
var BLACK = 16;

var PIECE_MASK = 7;
var COLOR_MASK = 24;

var PIECES = [
    " ", "♟", "♞", "♝", "♜", "♛", "♚", "?",
    " ", "♙", "♘", "♗", "♖", "♕", "♔", "?",
];

var requestAnim = window.requestAnimationFrame ||
    window.webkitRequestAnimationFrame ||
    window.mozRequestAnimationFrame ||
    window.oRequestAnimationFrame ||
    window.msRequestAnimationFrame ||
    function (callback) { window.setTimeout(callback, 1000 / 60); };


function ChessGame(canvas, clocks, addr) {
    this.clocks = clocks;
    this.clocks_ctx = clocks.getContext("2d");
    this.color = 0;
    this.turn = 0;
    this.board = [];
    this.sel = null;
    for (var i = 0; i < 64; i++)
        this.board[i] = 0;
    this.totalTime = 0;
    this.remainingA = 0;
    this.remainingB = 0;
    this.moves = [];
    this.anim = {};
    this.size = 400 / 9.0;

    this.game = document.getElementById("game");
    this.ctx_base = document.getElementById("base").getContext("2d");
    this.ctx_mark = document.getElementById("mark").getContext("2d");
    this.ctx_anim = document.getElementById("anim").getContext("2d");

    this.renderBase();
    this.renderClocks();

    var _this = this;

    if ('WebSocket' in window) {
        this.ws = new WebSocket(addr);
        this.ws.onopen = function(e) {
            document.getElementById("dlg-connect").style.display = 'none';
            document.getElementById("dlg-waiting").style.display = 'block';
        }
        this.ws.onmessage = function(e) {
            _this.process(e);
        };
        this.ws.onclose = function(e) {
            if (document.getElementById("dlg-result").style.display == 'none') {
                document.getElementById("dlg-waiting").style.display = 'none';
                document.getElementById("dlg-connect").style.display = 'none';
                document.getElementById("result").innerHTML = "Connection lost";
                document.getElementById("dlg-result").style.display = 'block';
            }
            _this.color = 0;
        };
        this.ws.onerror = function(e) {
            document.getElementById("dlg-result").style.display = 'none';
            document.getElementById("dlg-waiting").style.display = 'none';
            document.getElementById("dlg-connect").style.display = 'none';
            document.getElementById("result").innerHTML = "Connection error";
            document.getElementById("dlg-result").style.display = 'block';
            _this.color = 0;
        }
    } else {
        document.getElementById("connect").innerHTML = "Missing WebSocket Support";
    }
    this.game.addEventListener('click', function(e) {
        _this.click(e)
    });

    this.clocks_int = window.setInterval(function() {
        _this.tick();
    }, 1000);

    window.onbeforeunload = function(e) {
        if (_this.color != 0) {
            return "Leaving the page will cancel the current game.";
        }
    };
};


ChessGame.prototype.renderBase = function() {
    var ctx = this.ctx_base;
    var size = this.size;

    ctx.fillStyle = "#6288b9";
    ctx.fillRect(0, 0, 9*size, 9*size);

    ctx.font = 'bold 10pt "Helvetica Neue", Helvetica, Arial, sans-serif';
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillStyle = "#FFFFFF";
    if (this.color != 0) {
        for (var i = 0; i < 8; i++) {
            var rank = this.color == WHITE ? 7-i : i;
            var file = this.color == WHITE ? i : 7-i;
            ctx.fillText(rank+1, 0.25*size, (i+1)*size);
            ctx.fillText(rank+1, 8.75*size, (i+1)*size);
            ctx.fillText(String.fromCharCode(file+97), (i+1)*size, 0.25*size);
            ctx.fillText(String.fromCharCode(file+97), (i+1)*size, 8.75*size);
        }
    }

    for (var sq = 0; sq < 64; sq++) {
        this.renderBaseSq(sq);
    }
};

ChessGame.prototype.renderBaseSq = function(sq) {
    var ctx = this.ctx_base;
    var size = this.size;

    var x = (this.color != BLACK) ? sq&7 : 7-(sq&7);
    var y = (this.color != BLACK) ? 7-(sq>>3) : sq>>3;
    ctx.fillStyle = ((x&1) == (y&1)) ? "#FEFEFE" : "#83A5D2";
    ctx.fillRect(0.5*size+x*size, 0.5*size+y*size, size, size);

    ctx.font = '26pt "Helvetica Neue", Helvetica, Arial, sans-serif';
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillStyle = (sq == this.sel) ? "#FF0000" : "#000000";
    ctx.fillText(PIECES[this.board[sq]&15], (x+1)*size, (y+1)*size);
};

ChessGame.prototype.renderMarkers = function(src, moves) {
    var ctx = this.ctx_mark;
    var size = this.size;

    ctx.clearRect(0, 0, size*9, size*9);
    if (moves) {
        ctx.font = '26pt "Helvetica Neue", Helvetica, Arial, sans-serif';
        ctx.textAlign = "center";
        ctx.textBaseline = "middle";
        for (var i = 0; i < moves.length; i++) {
            var sq = moves[i];
            var x = (this.color == WHITE) ? sq&7 : 7-(sq&7);
            var y = (this.color == WHITE) ? 7-(sq>>3) : sq>>3;

            ctx.fillStyle = "rgba(60, 60, 60, 0.2)";
            var p = PIECES[this.board[src]&15];
            if (this.board[sq] != 0) {
                ctx.fillStyle = "rgba(255, 0, 0, 0.6)";
                p = "✘"
            }
            ctx.fillText(p, (x+1)*size, (y+1)*size);
        }
    }
};

ChessGame.prototype.renderAnim = function(t) {
    var ctx = this.ctx_anim;
    var size = this.size;

    ctx.clearRect(0, 0, size*9, size*9);

    ctx.font = '26pt "Helvetica Neue", Helvetica, Arial, sans-serif';
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillStyle = "#000000";

    var t = Date.now();
    var update = false;
    for (var a in this.anim) {
        var an = this.anim[a];
        var p = (t - an.t0) / (an.t1 - an.t0);
        if (p >= 1.0) {
            delete this.anim[a];
            p = 1.0;
            this.renderBaseSq(a);
        }
        ctx.fillText(PIECES[this.board[a]&15],
            an.x0+(an.x1-an.x0)*p,
            an.y0+(an.y1-an.y0)*p);
        update = true;
    }

    if (update) {
        var _this = this;
        requestAnim(function(t) {_this.renderAnim(t)});
    }
};

ChessGame.prototype.movePiece = function(src, dst) {
    var first = true;
    for (a in this.anim) {
        first = false;
        break;
    }

    var size = this.size;
    var now = Date.now();
    var dist = Math.sqrt(((src&7)-(dst&7))*((src&7)-(dst&7))+
        ((src>>3)-(dst>>3))*((src>>3)-(dst>>3)));

    this.anim[dst] = {
        t0: now,
        t1: now+150*dist,
        x0: (this.color == WHITE ? (1+(src&7))*size : (8-(src&7))*size),
        y0: (this.color == WHITE ? (8-(src>>3))*size : (1+(src>>3))*size),
        x1: (this.color == WHITE ? (1+(dst&7))*size : (8-(dst&7))*size),
        y1: (this.color == WHITE ? (8-(dst>>3))*size : (1+(dst>>3))*size),
    };

    this.board[dst] = this.board[src];
    this.board[src] = 0;
    this.renderBaseSq(src);

    if (first) {
        var _this = this;
        requestAnim(function(t) {_this.renderAnim(t)});
    }
};

ChessGame.prototype.renderClocks = function() {
    this.renderClock(0, 0, 130,
        this.totalTime > 0 ? this.remainingA / this.totalTime : 0, WHITE);
    this.renderClock(150, 0, 130,
        this.totalTime > 0 ? this.remainingB / this.totalTime : 0, BLACK);
};


ChessGame.prototype.renderClock = function(x, y, size, t, color) {
    var ctx = this.clocks_ctx;
    var active = false;
    if (this.color != 0) {
        active = (this.turn % 2 == 1) == (color == WHITE);
    }

    ctx.strokeStyle = "#cacad1";
    ctx.fillStyle = "#fafafa";
    ctx.lineWidth = 12;
    ctx.beginPath();
    ctx.arc(x+0.5*size, y+0.5*size, 0.45*size, 0, Math.PI*2, true);
    ctx.closePath();
    ctx.stroke();
    ctx.fill();

    ctx.globalCompositeOperation = "destination-out";
    ctx.beginPath();
    ctx.arc(x+0.5*size, y+0.5*size, 0.4*size, 0, Math.PI, true);
    ctx.arc(x+0.5*size, y+0.5*size, 0.1*size, Math.PI, 0, false);
    ctx.closePath();
    ctx.fill();
    ctx.globalCompositeOperation = "source-over";

    /* draw label */
    ctx.fillStyle = "#888";
    ctx.font = 'bold 12pt "Helvetica Neue", Helvetica, Arial, sans-serif';
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText(color == WHITE ? "white" : "black", x+0.5*size, y+0.7*size);

    /* draw pointer */
    ctx.fillStyle = active ? "#ee0000" : "#222";
    ctx.strokeStyle = active ? "#ee0000" : "#222";
    ctx.lineWidth = 2;
    ctx.beginPath();
    ctx.arc(x+0.5*size, y+0.5*size, 0.04*size,
        t*2*Math.PI-0.3*Math.PI, t*2*Math.PI-0.7*Math.PI, false);
    ctx.lineTo(x+0.5*size+0.475*size*Math.sin(t*2*Math.PI),
        y+0.5*size-0.475*size*Math.cos(t*2*Math.PI));
    ctx.closePath();
    ctx.fill();
    ctx.stroke();
}


ChessGame.prototype.click = function(e) {
    /* calculate the relative x and y position in pixels */
    var x, y;
    if (e.pageX != undefined && e.pageY != undefined) {
        x = e.pageX;
        y = e.pageY;
    } else {
        x = e.clientX + document.body.scrollLeft +
            document.documentElement.scrollLeft;
        y = e.clientY + document.body.scrollTop +
            document.documentElement.scrollTop;
    }
    x -= this.game.offsetLeft;
    y -= this.game.offsetTop;

    /* convert to field coordinates */
    var size = this.size;
    x = Math.floor((x - 0.5*size) / size);
    y = 7-Math.floor((y - 0.5*size) / size);
    if (this.color == BLACK) {
        x = 7 - x;
        y = 7 - y;
    }
    var pos = y*8+x;
    var prev_sel = this.sel;

    /* process the mouse click */
    if (x < 0 || x > 7 || y < 0 || y > 7 || this.sel == pos) {
        this.sel = null;
    } else if ((this.board[pos]&COLOR_MASK) == this.color) {
        this.sel = pos;
        this.ws.send(JSON.stringify({cmd: "select", turn: this.turn, src: pos}));
    } else if (this.sel != null && (this.turn % 2 == 1) == (this.color == WHITE)) {
        this.ws.send(JSON.stringify({cmd: "move", turn: this.turn, src: this.sel,
            dst: pos}));
        this.sel = null;
    }

    if (this.sel != prev_sel) {
        this.renderMarkers(this.sel, []);
        if (prev_sel != null) this.renderBaseSq(prev_sel);
        if (this.sel != null) this.renderBaseSq(this.sel);
    }
}


ChessGame.prototype.process = function(e) {
    var msg = JSON.parse(e.data);

    if (msg.cmd == "move") {
        if (this.board[msg.dst] == 0 && (msg.src&7) != (msg.dst&7)) {
            if (this.board[msg.src] == (P|WHITE)) {
                this.board[msg.dst-8] = 0;
                this.renderBaseSq(msg.dst-8);
            } else if (this.board[msg.src] == (P|BLACK)) {
                this.board[msg.dst+8] = 0;
                this.renderBaseSq(msg.dst+8);
            }
        }
        if (this.board[msg.src]==(K|WHITE) && msg.src==4) {
            if (msg.dst == 6) {
                this.movePiece(7, 5);
            } else if (msg.dst == 2) {
                this.movePiece(0, 3);
            }
        } else if (this.board[msg.src]==(K|BLACK) && msg.src==60) {
            if (msg.dst == 62) {
                this.movePiece(63, 61);
            } else if (msg.dst == 58) {
                this.movePiece(56, 59);
            }
        }
        this.movePiece(msg.src, msg.dst);
        if ((this.board[msg.dst] == (P|WHITE)) && (msg.dst>>3) == 7) {
            this.board[msg.dst] = (Q|WHITE);
        }
        if ((this.board[msg.dst] == (P|BLACK)) && (msg.dst>>3) == 0) {
            this.board[msg.dst] = (Q|BLACK);
        }
        this.turn = msg.turn + 1;
        this.remainingA = msg.RemainingA;
        this.remainingB = msg.RemainingB;
        this.renderClocks();
        if (msg.color == WHITE) {
            document.getElementById("history").innerHTML +=
                Math.floor((msg.turn+1)/2) + ".&nbsp;" + msg.History + "&nbsp;";
        } else {
            document.getElementById("history").innerHTML +=
                msg.History + " ";
        }
    }
    else if (msg.cmd == "start") {
        document.getElementById("dlg-waiting").style.display = 'none';
        this.board = [
            R|WHITE, N|WHITE, B|WHITE, Q|WHITE,
            K|WHITE, B|WHITE, N|WHITE, R|WHITE,
            P|WHITE, P|WHITE, P|WHITE, P|WHITE,
            P|WHITE, P|WHITE, P|WHITE, P|WHITE,
            0, 0, 0, 0, 0, 0, 0, 0,
            0, 0, 0, 0, 0, 0, 0, 0,
            0, 0, 0, 0, 0, 0, 0, 0,
            0, 0, 0, 0, 0, 0, 0, 0,
            P|BLACK, P|BLACK, P|BLACK, P|BLACK,
            P|BLACK, P|BLACK, P|BLACK, P|BLACK,
            R|BLACK, N|BLACK, B|BLACK, Q|BLACK,
            K|BLACK, B|BLACK, N|BLACK, R|BLACK,
        ];
        this.color = msg.color;
        this.turn = msg.turn;
        this.totalTime = msg.RemainingA;
        this.remainingA = msg.RemainingA;
        this.remainingB = msg.RemainingB;
        this.renderBase();
        this.renderClocks();
    }
    else if (msg.cmd == "msg") {
        document.getElementById("result").innerHTML = msg.Text;
        document.getElementById("dlg-result").style.display = "block";
        this.color = 0;
    }
    else if (msg.cmd == "ping") {
        this.ws.send(JSON.stringify({cmd: "pong"}));
    }
    else if (msg.cmd == "stat") {
        document.getElementById("numPlayers").innerHTML = msg.NumPlayers;
    }
    else if (msg.cmd == "select" && msg.src == this.sel) {
        this.renderMarkers(msg.src, msg.moves);
    }
}

ChessGame.prototype.tick = function() {
    if (this.color != 0) {
        if (this.turn%2 == 1) {
            this.remainingA -= 1000000000;
            if (this.remainingA < 0)
                this.remainingA = 0;
        } else {
            this.remainingB -= 1000000000;
            if (this.remainingB < 0)
                this.remainingB = 0;
        }
    }
    this.renderClocks();
}
