"use strict";

class Signaller {
    constructor(server, handle) {
        this.server = server;
        this.handle = handle;

        this.onsignal = null;
        this.onclose = null;

        this.__ws = null;
        this.__buffer = [];
    }

    listen() {
        this.__ws = new WebSocket(this.server + "/listen/" + this.handle);
        this.__setup_ws();
    }

    connect() {
        this.__ws = new WebSocket(this.server + "/connect/" + this.handle);
        this.__setup_ws();
    }

    signal(data) {
        if (this.__ws.readyState === 0)  // Socket is CONNECTING
            this.__buffer.push(data);
        else if (this.__ws.readyState === 1)  // Socket is OPEN
            this.__ws.send(data);
        else if (this.__ws.readyState === 2)
            throw new Error("Attempted to signal() a CLOSING WebSocket");
        else
            throw new Error("Attempted to signal() a CLOSED WebSocket");
    }

    close() {
        this.__ws.close(1000);  // 1000: Normal Closure
    }

    __setup_ws() {
        this.__ws.onopen = () => {
            this.__buffer.forEach((data) => this.__ws.send(data));
        };
        this.__ws.onmessage = (ev) => {
            this.onsignal(ev.data);
        };
        this.__ws.onerror = (ev) => {
            console.log("Signaller: WebSocket Error:", ev);
        };
        this.__ws.onclose = (ev) => {
            if (!this.onclose)
                return;

            this.onclose(ev);

        };
    }
}
