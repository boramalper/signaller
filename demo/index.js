const
    signaller = document.querySelector("#signaller"),
    handle = document.querySelector("#handle"),
    listen = document.querySelector("#listen"),
    connect = document.querySelector("#connect"),
    log = document.querySelector("#log"),
    send = document.querySelector("#msgbox button"),
    msg = document.querySelector("#msg")
;

const spConfig = {
    iceServers: [
        { urls: 'stun:stun.l.google.com:19302' },
        { urls: 'stun:global.stun.twilio.com:3478?transport=udp' }
    ]
};

// Signaller and SimplePeer
let si = null, sp = null;

function start(doWhat) {
    si = new Signaller(signaller.value, handle.value);
    sp = new SimplePeer({
        initiator: doWhat === "connect",
        config: spConfig,
    });

    si.onsignal = (data) => {
        log.innerHTML += span("Signaller received signal.");
        sp.signal(JSON.parse(data));
    };
    si.onclose = (ev) => {
        log.innerHTML += span("Signaller closed with code " + ev.code + " (" + ev.reason +").");
        si = null;
    };
    if (doWhat === "listen") si.listen();
    else                     si.connect();

    sp.on('error', (err) => {
        log.innerHTML += span("SimplePeer " + err);
        msg.setAttribute("disabled", "");
        send.setAttribute("disabled", "");
    });
    sp.on('signal', (data) => {
        // The WebSocket connection of the signaller might have failed (and thus closed).
        // Close SimplePeer if that's the case.
        if (si === null) {
            sp.destroy();
            return;
        }

        log.innerHTML += span("Signaller sends signal.");
        si.signal(JSON.stringify(data));
    });
    sp.on('connect', () => {
        log.innerHTML += span("Peer-to-Peer connection established, yay!");

        si.close();

        // Enable message box
        send.removeAttribute("disabled");
        msg.removeAttribute("disabled");
    });
    sp.on('data', function (data) {  // Remote peer messaged us!
        log.innerHTML += span("&gt; " + data);
        // Scroll log to the bottom.
        log.scrollTop = log.scrollHeight;
    });

    signaller.setAttribute("disabled", "");
    handle.setAttribute("disabled", "");
    listen.setAttribute("disabled", "");
    connect.setAttribute("disabled", "");

    if (doWhat === "listen")
        log.innerHTML += span("Signaller is listening on " + handle.value);
    else
        log.innerHTML += span("Signaller is connecting to " + handle.value);
}

function doSend() {
    sp.send(msg.value);
    log.innerHTML += span("&lt; " + msg.value);
    msg.value = "";
    log.scrollTop = log.scrollHeight;
}

function random(n) {
    let text = "";
    let possible = "abcdefghijklmnopqrstuvwxyz";

    for (let i = 0; i < n; i++)
        text += possible.charAt(Math.floor(Math.random() * possible.length));

    return text;
}

function span(txt) {
    const ts = new Date().toLocaleTimeString("tr-TR", {hour: "2-digit", minute: "2-digit", second: "2-digit"});
    return "<span><b>" + ts + "│</b> " + txt + "</span>";
}

(() => {
    handle.setAttribute("value", "random_" + random(10));
    if (window.location.hostname === "signaller.cecibot.com")
        signaller.setAttribute("value", "wss://" + window.location.hostname);
    else if (window.location.hostname)
        signaller.setAttribute("value", "ws://" + window.location.hostname + ":8080");
    else
        signaller.setAttribute("value", "ws://" + "127.0.0.1" + ":8080");

    document.querySelector("#msgbox").addEventListener("submit", (ev) => ev.preventDefault());
})();
