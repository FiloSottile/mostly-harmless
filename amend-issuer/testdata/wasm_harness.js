// Manual harness: drives the committed amend.wasm.gz through Node to verify the
// WebAssembly build and JS interface. Not part of `go test`.
//   node testdata/wasm_harness.js   (run from the package root)
globalThis.require = require;
globalThis.fs = require("fs");
const zlib = require("zlib");
require("../wasm_exec.js"); // defines globalThis.Go
const go = new Go();
const bytes = zlib.gunzipSync(fs.readFileSync("amend.wasm.gz"));
WebAssembly.instantiate(bytes, go.importObject).then((result) => {
  go.run(result.instance); // registers globalThis.amendIssuer, then blocks
  const dir = "testdata/stm-tpm-ecc";
  const issuer = fs.readFileSync(dir + "/stmtpmeccroot01.pem", "utf8");
  const child = fs.readFileSync(dir + "/stmtpmeccint02.pem", "utf8");

  const out = globalThis.amendIssuer(issuer, child);
  if (out.error) {
    console.error("unexpected error:", out.error);
    process.exit(1);
  }
  const expected = fs.readFileSync(dir + "/amended.pem", "utf8");
  if (out.pem !== expected) {
    console.error("MISMATCH\n--- got ---\n" + out.pem + "\n--- want ---\n" + expected);
    process.exit(1);
  }
  console.log("WASM happy path matches expected.pem");

  const swapped = globalThis.amendIssuer(child, issuer);
  if (!swapped.error) {
    console.error("expected error on swapped inputs");
    process.exit(1);
  }
  console.log("WASM swapped-inputs error:", JSON.stringify(swapped.error));

  const garbage = globalThis.amendIssuer("not a pem", issuer);
  if (!garbage.error) {
    console.error("expected error on garbage input");
    process.exit(1);
  }
  console.log("WASM garbage-input error:", JSON.stringify(garbage.error));

  process.exit(0);
});
