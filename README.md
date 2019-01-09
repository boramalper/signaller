# signaller
_A WebRTC signalling server in Go using WebSockets._

[![Build Status](https://travis-ci.org/boramalper/signaller.svg?branch=master)](https://travis-ci.org/boramalper/signaller)

**signaller** consists of two components:
- **signallerd**, the WebRTC signalling server using WebSockets.
- **signaller.js**, a helper JavaScript library.

## Quick Start
1. Generate a random handle and open a WebSocket to
   `ws://example.com/listen/<handle>`.
2. The peer that you want to connect with must open a WebSocket
   to `ws://example.com/connect/<handle>`.
3. Now you can exchange WebRTC signals with each other using the WebSocket
   you opened; your messages will be relayed through the **signaller** server at `example.com`.

### Handles
- Handles are at least 3 at most 32 characters long.
- Handles begin with a lowercase character.
- Handles can contain lowercase characters and underscores.
- Handles cannot contain multiple consecutive underscores.

## signaller.js
**signaller.js** is a tiny, client-side JavaScript library over WebSocket API that buffers
your signals until the underlying WebSocket is ready.

### signaller.js Usage
1. Create a `Signaller` object by providing a `server` and a `handle`.

   ```javascript
   let signaller = new Signaller("ws://example.com", "a_handle")
   ```
   
2. Call either one of `.listen()` or `.connect()` methods, depending on whether you
   are waiting for another peer, or whether you are connecting to a peer that awaits.
   
   ```javascript
   signaller.listen();
   // OR signaller.connect();
   ```
   
3. Set an `onsignal` event handler, where you handle the incoming signals.

   ```javascript
   signaller.onsignal = (data) => {
       // Handle the incoming signal (data)
   }
   ``` 
   
4. Send your signals using `.signal()` method.

   ```javascript
   signaller.signal(mySignalData);
   ```

5. Close the signaller using `.close()` method.

   ```javascript
   signaller.close();
   ```

Also see [the demo](demo), using [simple-peer](https://github.com/feross/simple-peer) by [Feross Aboukhadijeh](http://feross.org/).

### Closure Codes
- **`1000` - Normal Closure**

  Normal closure; the connection successfully completed whatever purpose for
  which it was created.
  
- **`1006` - Abnormal Closure**

  Used to indicate that a connection was closed abnormally (that is, with no
  close frame being sent) when a status code is expected.
  
  This should never occur, please file a bug report!
  
- **`1009` - Message Too Big**

  The server is terminating the connection because a data frame was received
  that is too large.
  
- **`1011` - Internal Error**

  The server is terminating the connection because it encountered an unexpected
  condition that prevented it from fulfilling the request.
  
- **`4000` - Listen Timeout**

  Timed out while listening.
  
- **`4001` - Pipe Timeout**
  
  Timed out while piping data.
  
  You should exchange WebRTC signalling data and *close* the WebSocket as soon
  as possible  to prevent Pipe Timeout. 
  
- **`4002` - Too Many Messages**

  Too many messages in total have been sent by either one of the peers.

## signallerd
**signallerd** is a WebSockets relay written in Go. It is small, around 200 lines, and it relies on
standards compliant [gorilla/websocket](https://github.com/gorilla/websocket#gorilla-websocket-compared-with-other-packages)
package.

**signallerd** is highly configurable:

- You can supply a regex to match the Origin HTTP request header against
  - ...if you'd like to prevent others from using your resources.
- You can set deadlines on
  - listening sockets (*i.e.* WebSockets that wait for a remote peer), and
  - on whole "pipes" (*i.e.* the pair of linked WebSockets)
- You can set the maximum message size
- You can set the maximum number of messages that can be sent from a peer.

You can always run `signallerd -help` for command-line help.

One important downside of **signallerd** is that it currently doesn't have any rate-limiting
capabilities, neither for bandwidth nor for requests (based on IP addresses).

## License
ISC License, see [LICENSE](./LICENSE).
