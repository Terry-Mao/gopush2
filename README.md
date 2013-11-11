## Terry-Mao/gopush2
`Terry-Mao/gopush` is an push server written by golang. (websocket)

## Requeriments
```sh
# all platform
# websocket
$ go get code.google.com/p/go.net/websocket 
```

## Installation && Update
Just pull `Terry-Mao/gopush` from github using `go get`:

```sh
$ go get -u github.com/Terry-Mao/gopush
```

## Usage
```sh
# start the gopush server
$ ./gopush2 -c ./gopush2.conf

# 1. open http://localhost:8080/client in browser and press the Send button (modify the gopush2.conf, set debug to 1)
# 2. you can use curl
$ curl -d "test" http://localhost:8080/pub?key=Terry-Mao\&expire=30
# then your browser will alert the "message"
```
a simple java client example
```java
import java.io.IOException;
import java.util.concurrent.ExecutionException;

import com.ning.http.client.*;
import com.ning.http.client.websocket.*;

public class test {

	/**
	 * @param args
	 * @throws IOException
	 * @throws ExecutionException
	 * @throws InterruptedException
	 */
	public static void main(String[] args) throws InterruptedException,
			ExecutionException, IOException {

		AsyncHttpClient c = new AsyncHttpClient();

		WebSocket websocket = c
				.prepareGet("ws://10.33.30.66:8080/sub?key=Terry-Mao&mid=0")
				.execute(
						new WebSocketUpgradeHandler.Builder()
								.addWebSocketListener(
										new WebSocketTextListener() {

											@Override
											public void onMessage(String message) {
												System.out.println(message);
											}

											@Override
											public void onOpen(
													WebSocket websocket) {
												System.out.println("ok");
											}

											public void onClose(
													WebSocket websocket) {
											}

											@Override
											public void onError(Throwable t) {
											}

											@Override
											public void onFragment(String arg0,
													boolean arg1) {
												// TODO Auto-generated method
												// stub

											}
										}).build()).get();

		Thread.sleep(100000000);
	}
}

```

## Protocol
response json field:
### msg
* the publish message

### msg_id
* the publish message id

the reponse json examples:
```json
{
    "msg" : "hello, world",
    "msg_id" : 1
}
```

## Documentation
Read the `Terry-Mao/gopush2` documentation from a terminal
```sh
$ go doc github.com/Terry-Mao/gopush2
```

Alternatively, you can [gopush2](http://go.pkgdoc.org/github.com/Terry-Mao/gopush2).
