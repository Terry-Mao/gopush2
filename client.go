package main

import (
	"fmt"
	"io"
	"net/http"
)

func Client(w http.ResponseWriter, r *http.Request) {
	addr := Conf.Addr
	if addr == "" {
		addr = "localhost"
	}

	html := fmt.Sprintf(`
<!doctype html>
<html>

    <script type="text/javascript" src="http://img3.douban.com/js/packed_jquery.min6301986802.js" async="true"></script>
    <script type="text/javascript">
        var sock = null;
        var wsuri = "ws://%s:%d/sub?key=Terry-Mao&mid=0&token=test";

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
                if(e.data != "") {
                    alert("message received: " + e.data);
                }
            }

        };

        setInterval("sock.send('')", 3000);
</script>
<h1>Push Service </h1>
`, addr, Conf.Port)
	io.WriteString(w, html)
}
