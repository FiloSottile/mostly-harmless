// Loads the Go WebAssembly module and wires up the form. wasm_exec.js (which
// defines the global Go) must be loaded before this script.
//
// The module is committed gzip-compressed (amend.wasm.gz) and decompressed in
// the browser, to keep the repository small. Assets are fetched with relative
// URLs so the page works both when served from its own directory and through
// htmlpreview.github.io, which injects a <base> tag pointing at the original
// directory.

(function () {
    "use strict";

    var issuer = document.getElementById("issuer");
    var child = document.getElementById("child");
    var button = document.getElementById("amend");
    var output = document.getElementById("output");
    var copy = document.getElementById("copy");

    function show(text, kind) {
        output.textContent = text;
        output.className = kind || "";
        // Copying only makes sense for a successful result (kind === "").
        copy.disabled = kind !== "";
        copy.textContent = "Copy";
    }

    // wasmBytes fetches amend.wasm.gz and gunzips it. If the host transparently
    // decompressed the response (Content-Encoding), the bytes are used as-is.
    function wasmBytes() {
        return fetch("amend.wasm.gz")
            .then(function (resp) {
                if (!resp.ok) throw new Error("HTTP " + resp.status);
                return resp.arrayBuffer();
            })
            .then(function (buffer) {
                var bytes = new Uint8Array(buffer);
                var gzipped = bytes[0] === 0x1f && bytes[1] === 0x8b;
                if (!gzipped) return buffer;
                if (typeof DecompressionStream !== "function") {
                    throw new Error("this browser cannot decompress amend.wasm.gz");
                }
                var stream = new Response(bytes).body.pipeThrough(new DecompressionStream("gzip"));
                return new Response(stream).arrayBuffer();
            });
    }

    var go = new Go();
    wasmBytes()
        .then(function (bytes) {
            return WebAssembly.instantiate(bytes, go.importObject);
        })
        .then(function (result) {
            go.run(result.instance); // registers globalThis.amendIssuer, then blocks
            button.disabled = false;
            button.textContent = "Amend issuer";
        })
        .catch(function (err) {
            button.textContent = "Failed to load";
            show("Failed to load the WebAssembly module: " + err.message, "error");
        });

    button.addEventListener("click", function () {
        var result = globalThis.amendIssuer(issuer.value, child.value);
        if (result.error) {
            show(result.error, "error");
        } else {
            show(result.pem, "");
        }
    });

    copy.addEventListener("click", function () {
        var text = output.textContent;
        function copied() {
            copy.textContent = "Copied!";
            setTimeout(function () { copy.textContent = "Copy"; }, 1500);
        }
        function selectFallback() {
            var range = document.createRange();
            range.selectNodeContents(output);
            var selection = window.getSelection();
            selection.removeAllRanges();
            selection.addRange(range);
            try {
                document.execCommand("copy");
                copied();
            } catch (e) { /* leave the text selected for manual copy */ }
        }
        if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(text).then(copied, selectFallback);
        } else {
            selectFallback();
        }
    });
})();
