package main

import (
	"fmt"
	"io"
	"net/http"
)

func Client(w http.ResponseWriter, r *http.Request) {
	addr := Conf.Addr
	if addr == "" {
		addr = "localhost:8080"
	}

	html := fmt.Sprintf(`
<!doctype html>
<html>

    <script type="text/javascript" src="http://img3.douban.com/js/packed_jquery.min6301986802.js" async="true"></script>
    <script type="text/javascript">
        var sock = null;
        var wsuri = "ws://%s/sub?key=Terry-Mao&mid=0&token=test";
        var first = true;

        window.onload = function() {
            try
            {
                sock = new WebSocket(wsuri);
            }catch (e) {
                alert(e.Message);
            }

            sock.onopen = function() {
                alert("connected to " + wsuri);
            }

            sock.onerror = function(e) {
                alert(" error from connect " + e.Message);
            }

            sock.onclose = function(e) {
                alert("connection closed (" + e.code + ")");
            }

            sock.onmessage = function(e) {
                if(e.data != "h") {
                    alert("message received: " + e.data);
                } else if(e.data == "h") {
                    if(first) {
                        setInterval("sock.send('h')", 3000);
                        first = false;
                    }
                }
            }

        };
</script>
<h1>Push Service </h1>
`, addr)
	io.WriteString(w, html)
}
