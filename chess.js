/**
 * ChessBuddy - Play chess with Go, HTML5, WebSockets and random strangers
 *
 * Copyright (c) 2012 by Christoph Hack <christoph@tux21b.org>
 * All rights reserved. Distributed under the Simplified BSD License.
 */

var SYMBOLS = ["♟", "♞", "♝", "♜", "♛", "♚", " ", "♔", "♕", "♖", "♗", "♘", "♙"];


function ChessGame(canvas, clocks, websocket) {
    this.canvas = canvas;
    this.canvas_ctx = canvas.getContext("2d");
    this.clocks = clocks;
    this.clocks_ctx = clocks.getContext("2d");
    this.color = 0;
    this.turn = 0;
    this.board = [];
    this.sel = null;
    for (var i = 0; i < 64; i++)
        this.board[i] = 0;
    this.ws = websocket;
    this.status = 0;

    var _this = this;
    this.ws.onmessage = function(e) { _this.process(e); };
    this.ws.onclose = function(e) { _this.status = 5; _this.render(); };
    this.canvas.addEventListener('click', function(e) {
        _this.click(e)
    });
    this.render();
}


ChessGame.prototype.render = function() {
    var ctx = this.canvas_ctx;
    var size = Math.min(this.canvas.width, this.canvas.height) / 9.0;
    var border = 0.5*size;

    /* draw the checker board */
    ctx.fillStyle = "#6288b9";
    ctx.fillRect(0, 0, 9*size, 9*size);
    for (var y = 0; y < 8; y++) {
        for (var x = 0; x < 8; x++) {
            ctx.fillStyle = ((x & 1) != (y & 1)) ? "#83A5D2" : "#FEFEFE";
            ctx.fillRect(border+x*size, border+y*size, size, size);
        }
    }

    /* draw labels */
    ctx.font = 'bold 10pt "Helvetica Neue", Helvetica, Arial, sans-serif';
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillStyle = "#FFFFFF";
    if (this.color != 0) {
        for (var i = 0; i < 8; i++) {
            var row = this.color > 0 ? 7-i : i;
            var col = this.color > 0 ? i : 7-i;
            ctx.fillText(row+1, 0.25*size, (i+1)*size);
            ctx.fillText(row+1, 8.75*size, (i+1)*size);
            ctx.fillText(String.fromCharCode(col+97), (i+1)*size, 0.25*size);
            ctx.fillText(String.fromCharCode(col+97), (i+1)*size, 8.75*size);
        }
    }

    /* draw figures (incl. selection) */
    ctx.font = '26pt "Helvetica Neue", Helvetica, Arial, sans-serif';
    for (var y = 0; y < 8; y++) {
        for (var x = 0; x < 8; x++) {
            ctx.fillStyle = "#000000";
            var p = this.color > 0 ? (7-y)*8+x : y*8+7-x;
            if (this.sel != null && p == this.sel.y*8+this.sel.x) {
                ctx.fillStyle = "#ff0000"
            }
            ctx.fillText(SYMBOLS[this.board[p]+6],
                border+(x+0.5)*size, border+(y+0.5)*size);
        }
    }

    /* draw messages */
    var msg = "";
    if (this.color == 0) {
        msg = "Waiting for another player...";
    } else if (this.status == 5) {
        msg = "Opponent left... Reload?";
    }
    if (msg) {
        ctx.fillStyle = "rgba(220, 220, 220, 0.8)";
        ctx.fillRect(0, 3.75*size, 9*size, 1.5*size);
        ctx.fillStyle = "#000000";
        ctx.font = '20pt "Helvetica Neue", Helvetica, Arial, sans-serif';
        ctx.fillText(msg, 4.5*size, 4.5*size);
    }

    /* draw clocks */
    this.renderClock(0, 0, 130, 0, (this.color != 0) && ((this.turn & 1) == 1));
    this.renderClock(150, 0, 130, 0.25, (this.color != 0) && ((this.turn & 1) == 0));
};

ChessGame.prototype.renderClock = function(x, y, size, t, active) {
    var ctx = this.clocks_ctx;
    ctx.fillStyle = "#cccccc";
    ctx.beginPath();
    ctx.arc(x+0.5*size, y+0.5*size, 0.5*size, 0, Math.PI*2, true);
    ctx.closePath();
    ctx.fill();
    ctx.fillStyle = "#ffffff";
    ctx.beginPath();
    ctx.arc(x+0.5*size, y+0.5*size, 0.45*size, 0, Math.PI*2, true);
    ctx.closePath();
    ctx.fill();

    ctx.fillStyle = active ? "#ff0000" : "#000000";
    ctx.beginPath();
    ctx.arc(x+0.5*size, y+0.5*size, 0.05*size, 0, Math.PI*2, true);
    ctx.closePath();
    ctx.fill();

    ctx.strokeStyle = active ? "#ff0000" : "#000000";
    ctx.lineWidth = 8;
    ctx.beginPath();
    ctx.moveTo(x+0.5*size, y+0.5*size);
    ctx.lineTo(x+0.5*size+0.475*size*Math.sin(t*Math.PI),
        y+0.5*size-0.475*size*Math.cos(t*Math.PI));
    ctx.closePath();
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
    x -= this.canvas.offsetLeft;
    y -= this.canvas.offsetTop;

    /* convert to field coordinates */
    var size = Math.min(canvas.width, canvas.height) / 9.0;
    x = Math.floor((x - 0.5*size) / size);
    y = 7-Math.floor((y - 0.5*size) / size);
    if (this.color < 0) {
        x = 7 - x;
        y = 7 - y;
    }

    /* process the mouse click */
    if (x < 0 || x > 7 || y < 0 || y > 7 || (this.sel != null &&
        x == this.sel.x && y == this.sel.y)) {
        this.sel = null;
    } else if (this.board[y*8+x]*this.color > 0) {
        this.sel = {x: x, y: y};
    } else if (this.sel != null) {
        ws.send(JSON.stringify({Cmd: "move", Turn: this.turn,
            ax: this.sel.x, ay: this.sel.y, bx: x, by: y}));
        this.sel = null;
    }
    this.render();
}


ChessGame.prototype.process = function(e) {
    var msg = JSON.parse(e.data);

    if (msg.Cmd == "move") {
        this.board[msg.By*8+msg.Bx] = this.board[msg.Ay*8+msg.Ax];
        this.board[msg.Ay*8+msg.Ax] = 0;
        this.turn = msg.Turn+1;
        document.getElementById("history").innerHTML += msg.History + " ";
        this.render();
    }
    else if (msg.Cmd == "start") {
        this.board = msg.Board;
        this.color = msg.Color;
        this.turn = 1;
        this.render();
    }
    else if (msg.Cmd == "ping") {
        ws.send(JSON.stringify({Cmd: "pong"}));
    }
    document.getElementById("numPlayers").innerHTML = msg.NumPlayers;
}
