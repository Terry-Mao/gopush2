package main

import (
	"fmt"
	"io"
	"net/http"
)

func Client(w http.ResponseWriter, r *http.Request) {
	html := fmt.Sprintf(`
<!doctype html>
<html>

    <script type="text/javascript" src="http://img3.douban.com/js/packed_jquery.min6301986802.js" async="true"></script>
    <script type="text/javascript">
        var sock = null;
        var wsuri = "ws://%s:%d/sub?key=Terry-Mao&msg_id=0";

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
                alert("message received: " + e.data);
            }

        };
</script>
<h1>Push Service </h1>
`, Conf.Addr, Conf.Port)
	io.WriteString(w, html)
}
