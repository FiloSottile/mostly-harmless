// Drives the real index.html in headless Chrome over the DevTools Protocol:
// waits for the WASM to load, fills the textareas, clicks Amend issuer, and
// checks the output equals the golden amended.pem. Not part of `go test`.
//   node testdata/browser_drive.js <chrome-bin> <page-url>
const { spawn } = require("child_process");
const fs = require("fs");
const http = require("http");
const os = require("os");
const path = require("path");

const chromeBin = process.argv[2];
const pageURL = process.argv[3];
const dir = "testdata/stm-tpm-ecc";
const issuerPEM = fs.readFileSync(dir + "/stmtpmeccroot01.pem", "utf8");
const childPEM = fs.readFileSync(dir + "/stmtpmeccint02.pem", "utf8");
const expected = fs.readFileSync(dir + "/amended.pem", "utf8");

const port = 9237;
const userDataDir = fs.mkdtempSync(path.join(os.tmpdir(), "amend-chrome-"));
const chrome = spawn(chromeBin, [
  "--headless=new",
  "--disable-gpu",
  "--no-first-run",
  "--no-default-browser-check",
  "--remote-debugging-port=" + port,
  "--user-data-dir=" + userDataDir,
  pageURL,
]);

function cleanup(code) {
  try { chrome.kill("SIGKILL"); } catch {}
  try { fs.rmSync(userDataDir, { recursive: true, force: true }); } catch {}
  process.exit(code);
}

function fail(msg) {
  console.error("FAIL:", msg);
  cleanup(1);
}

function getJSON(url) {
  return new Promise((resolve, reject) => {
    http.get(url, (res) => {
      let data = "";
      res.on("data", (c) => (data += c));
      res.on("end", () => resolve(JSON.parse(data)));
    }).on("error", reject);
  });
}

async function findPageTarget() {
  for (let i = 0; i < 50; i++) {
    try {
      const targets = await getJSON(`http://127.0.0.1:${port}/json/list`);
      const page = targets.find((t) => t.type === "page" && t.webSocketDebuggerUrl);
      if (page) return page;
    } catch {}
    await new Promise((r) => setTimeout(r, 100));
  }
  throw new Error("no page target found");
}

(async () => {
  const page = await findPageTarget();
  const ws = new WebSocket(page.webSocketDebuggerUrl);
  let nextId = 1;
  const pending = new Map();
  ws.addEventListener("message", (ev) => {
    const msg = JSON.parse(ev.data);
    if (msg.id && pending.has(msg.id)) {
      pending.get(msg.id)(msg);
      pending.delete(msg.id);
    }
  });
  function send(method, params) {
    const id = nextId++;
    ws.send(JSON.stringify({ id, method, params }));
    return new Promise((resolve) => pending.set(id, resolve));
  }
  async function evaluate(expression) {
    const res = await send("Runtime.evaluate", {
      expression,
      returnByValue: true,
      awaitPromise: true,
    });
    if (res.result && res.result.exceptionDetails) {
      throw new Error("page exception: " + JSON.stringify(res.result.exceptionDetails));
    }
    return res.result.result.value;
  }

  await new Promise((resolve, reject) => {
    ws.addEventListener("open", resolve);
    ws.addEventListener("error", reject);
  });
  await send("Runtime.enable", {});
  await send("Browser.grantPermissions", {
    permissions: ["clipboardReadWrite", "clipboardSanitizedWrite"],
  });

  // Wait for the WASM to load: the button is enabled and relabeled.
  let ready = false;
  for (let i = 0; i < 100; i++) {
    const label = await evaluate(`(() => { const b = document.getElementById('amend'); return b ? (b.disabled ? '' : b.textContent) : null; })()`);
    if (label && label.trim() === "Amend issuer") { ready = true; break; }
    await new Promise((r) => setTimeout(r, 100));
  }
  if (!ready) fail("WASM did not load / button never enabled");
  console.log("button enabled after WASM load: OK");

  // Fill the form, click, and read the output (amendIssuer is synchronous).
  const expr = `(() => {
    document.getElementById('issuer').value = ${JSON.stringify(issuerPEM)};
    document.getElementById('child').value = ${JSON.stringify(childPEM)};
    document.getElementById('amend').click();
    const o = document.getElementById('output');
    return JSON.stringify({ text: o.textContent, cls: o.className });
  })()`;
  const out = JSON.parse(await evaluate(expr));
  if (out.cls === "error") fail("output marked as error: " + out.text);
  if (out.text !== expected) {
    fail("output mismatch\n--- got ---\n" + out.text + "\n--- want ---\n" + expected);
  }
  console.log("click produced expected amended PEM: OK");

  // Copy button: enabled after a result, and copies the output to the clipboard.
  const copyEnabled = await evaluate(`!document.getElementById('copy').disabled`);
  if (!copyEnabled) fail("copy button should be enabled after a result");
  await evaluate(`document.getElementById('copy').click()`);
  let copied = false;
  for (let i = 0; i < 30; i++) {
    const label = await evaluate(`document.getElementById('copy').textContent`);
    if (label === "Copied!") { copied = true; break; }
    await new Promise((r) => setTimeout(r, 50));
  }
  if (!copied) fail("copy button did not show 'Copied!' feedback");
  const clip = await evaluate(`navigator.clipboard.readText()`);
  if (clip !== expected) fail("clipboard contents do not match the output");
  console.log("copy button copies amended PEM to clipboard: OK");

  // Error path in-browser: garbage input shows an error.
  const errExpr = `(() => {
    document.getElementById('issuer').value = 'garbage';
    document.getElementById('amend').click();
    const o = document.getElementById('output');
    return JSON.stringify({ text: o.textContent, cls: o.className });
  })()`;
  const errOut = JSON.parse(await evaluate(errExpr));
  if (errOut.cls !== "error" || !errOut.text) fail("expected an error to be shown for garbage input");
  console.log("error path shows error in browser:", JSON.stringify(errOut.text));

  console.log("ALL BROWSER CHECKS PASSED");
  cleanup(0);
})().catch((e) => fail(e.message || String(e)));
