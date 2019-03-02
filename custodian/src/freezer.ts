const pageLoadTimeout = 60 * 1000 // milliseconds

import * as puppeteer from 'puppeteer'
import * as minimist from 'minimist'
import { JSDOM } from 'jsdom'

// https://github.com/ChromeDevTools/devtools-protocol/issues/106
import { Protocol as CDP } from 'devtools-protocol'

console = new console.Console(process.stderr, process.stderr)

const argv = minimist(process.argv.slice(2), { stopEarly: true })
if (argv._.length != 1) {
    console.error("Missing URL argument")
    process.exit(1)
}
const URL = argv._[0]

console.log("Starting...");
(async () => {
    const browser = await puppeteer.launch()
    await runWithContext(browser, async (page: puppeteer.Page, client: puppeteer.CDPSession) => {
        await freezePage(page, client, URL)
    })
    await browser.close()
})().catch((reason) => { console.error(reason) })

async function runWithContext(browser: puppeteer.Browser, f: (page: puppeteer.Page, client: puppeteer.CDPSession) => Promise<void>) {
    // https://github.com/DefinitelyTyped/DefinitelyTyped/issues/26626
    const context: puppeteer.Browser = await (browser as any).createIncognitoBrowserContext()
    const page = await context.newPage()
    page.on('console', msg => console.log("remote:", msg.text()))
    const client = await page.target().createCDPSession()
    await Promise.all([client.send('DOM.enable'), client.send('CSS.enable'), client.send('Page.enable')])
    await f(page, client).catch((reason) => { console.error(reason) })
    await client.detach()
    await context.close()
}

async function freezePage(page: puppeteer.Page, client: puppeteer.CDPSession, URL: string) {
    console.log("Fetching page " + URL + "...")
    console.time("fetch")

    await page.setViewport({ width: 1280, height: 850 })
    await page.goto(URL, {
        timeout: pageLoadTimeout,
        waitUntil: ["load", "networkidle0"],
        // https://github.com/GoogleChrome/puppeteer/issues/1353#issuecomment-356561654
    })
    await client.send("Page.stopLoading")

    // TODO: immediately unload all scripts, or anyway stop the page

    // Restore console.log, because fuck you Twitter.
    await page.evaluate(() => {
        var i = document.createElement('iframe')
        i.style.display = 'none'
        document.body.appendChild(i);
        (window as any).console = i.contentWindow!.console
        // i.parentNode!.removeChild(i)
    })

    console.timeEnd("fetch")
    console.time("sync processing")

    // Build up a virtual DOM for performance and control.
    const dom = new JSDOM()
    const document = dom.window.document

    async function getMatchedStyle(nodeId: CDP.DOM.NodeId): Promise<CDP.CSS.GetMatchedStylesForNodeResponse> {
        return client.send('CSS.getMatchedStylesForNode',
            { nodeId: nodeId } as CDP.CSS.GetMatchedStylesForNodeRequest)
    }
    async function getComputedStyleCDP(nodeId: CDP.DOM.NodeId): Promise<Map<string, string>> {
        var res = new Map<string, string>()
        const computedStyle: CDP.CSS.GetComputedStyleForNodeResponse = await client.send(
            'CSS.getComputedStyleForNode',
            { nodeId: nodeId } as CDP.CSS.GetComputedStyleForNodeRequest
        )
        for (var prop of computedStyle.computedStyle) {
            res.set(prop.name, prop.value)
        }
        return res
    }

    async function getStyledProperties(nodeId: CDP.DOM.NodeId): Promise<Set<string>> {
        var res = new Set<string>()
        const matchedStyle = await getMatchedStyle(nodeId)
        for (var rule of (matchedStyle.matchedCSSRules || [])) {
            if (rule.rule.origin == "user-agent") continue
            for (var prop of rule.rule.style.cssProperties) {
                res.add(prop.name)
            }
        }
        if (matchedStyle.inlineStyle) {
            for (var prop of matchedStyle.inlineStyle!.cssProperties) {
                res.add(prop.name)
            }
        }
        if (matchedStyle.attributesStyle) {
            for (var prop of matchedStyle.attributesStyle!.cssProperties) {
                res.add(prop.name)
            }
        }
        return res
    }

    async function inlineStyles(node: CDP.DOM.Node, el: HTMLElement): Promise<void> {
        const computedStyle = getComputedStyleCDP(node.nodeId)
        const styledProps = getStyledProperties(node.nodeId)

        var style = "";
        await Promise.all(Array.from(await styledProps).map(async (name) => {
            let computedValue = (await computedStyle).get(name)
            if (!computedValue) return
            if (name == "background-image") {
                // Because string.replace does not work with async/await, wtf...
                let res = ""
                let lastIndex = 0
                for (let re = /url\("([^"]+)"\)/g, match: RegExpExecArray | null;
                    match = re.exec(computedValue); match !== null) {
                    res += computedValue.slice(lastIndex, match.index)
                    lastIndex = match.index + match[0].length
                    const url = match[1]
                    const dataURL = await dataURLForResource(url)
                    if (dataURL === null) res += 'url("")'
                    else res += 'url("' + dataURL + '")'
                }
                res += computedValue.slice(lastIndex)
                computedValue = res
            }
            style += name + ":" + computedValue + ";"
        }))
        if (style != "") {
            el.setAttribute("style", style)
        }
    }

    async function getCurrentSrc(nodeId: CDP.DOM.NodeId): Promise<string | null> {
        // https://github.com/cyrus-and/chrome-remote-interface/issues/78
        // Could alternatively call Runtime.getProperties, but effectively it executes Javascript.
        const remoteObj: CDP.DOM.ResolveNodeResponse = await client.send('DOM.resolveNode', {
            nodeId: nodeId,
            objectGroup: "get-current-src", // TODO: consider releasing, but we use throw-away contexts anyway
        } as CDP.DOM.ResolveNodeRequest)
        const result: CDP.Runtime.CallFunctionOnResponse = await client.send('Runtime.callFunctionOn', {
            objectId: remoteObj.object.objectId!, silent: true, returnByValue: true, generatePreview: false,
            functionDeclaration: "function currentSrc() { return this.currentSrc; }",
        } as CDP.Runtime.CallFunctionOnRequest)
        if (result.result.type != "string") return null
        const currentSrc = result.result.value! as string
        if (currentSrc == "") return null
        return currentSrc
    }

    var a: HTMLAnchorElement
    function absoluteURL(url: string): string {
        if (!a) a = document.createElement("a")
        a.href = url
        return a.href
    }

    async function getResourceTree(): Promise<CDP.Page.FrameResourceTree> {
        const res = await client.send('Page.getResourceTree') as CDP.Page.GetResourceTreeResponse
        return res.frameTree
    }
    async function getResourceContent(frameId: CDP.Page.FrameId, url: string): Promise<string> {
        const res = await client.send('Page.getResourceContent', { frameId: frameId, url: url } as CDP.Page.GetResourceContentRequest) as CDP.Page.GetResourceContentResponse
        console.assert(res.base64Encoded)
        return res.content
    }

    var imgDataURLs = new Map<string, string>()
    const resTree = await getResourceTree()
    async function dataURLForResource(url: string): Promise<string | null> {
        for (var res of resTree.resources) {
            if (imgDataURLs.has(url)) return imgDataURLs.get(url)!
            if (res.url != url) continue
            if (res.type != "Image") continue
            if (res.failed || res.canceled) continue
            // TODO: based on res.contentSize, decide to make a separate file
            const b64Content = await getResourceContent(resTree.frame.id, res.url)
            const dataURL = "data:" + res.mimeType + ";base64," + b64Content
            imgDataURLs.set(url, dataURL)
            return dataURL
        }
        return null
    }

    async function setImageURL(node: CDP.DOM.Node, el: HTMLElement) {
        if (node.nodeName != "IMG") return
        const currentSrc = await getCurrentSrc(node.nodeId)
        if (currentSrc === null) return
        const dataURL = await dataURLForResource(currentSrc)
        if (dataURL === null) return
        el.setAttribute("src", dataURL)
    }

    function getLowerCaseAttribute(node: CDP.DOM.Node, name: string): string | null {
        if (node.attributes === undefined) return null
        for (let i = 0; i < node.attributes.length; i += 2) {
            if (node.attributes[i].toLowerCase() == name) {
                return node.attributes[i + 1].toLowerCase()
            }
        }
        return null
    }

    function shouldDropElement(node: CDP.DOM.Node, head: boolean): boolean {
        if (node.nodeName == "SCRIPT" || node.nodeName == "STYLE" || node.nodeName == "LINK") {
            return true
        }
        if (node.nodeName == "META" && getLowerCaseAttribute(node, "charset") !== null) {
            return false
        }
        if (head && node.nodeName != "TITLE" && node.nodeName != "HEAD") {
            return true
        }
        return false
    }

    async function applyAttributes(node: CDP.DOM.Node, el: HTMLElement) {
        if (node.attributes === undefined) return
        for (let i = 0; i < node.attributes.length; i += 2) {
            switch (node.attributes[i].toLowerCase()) {
                case "charset":
                case "href":
                case "datetime":
                    el.setAttribute(node.attributes[i], node.attributes[i + 1])
                    break
            }
        }
    }

    let parallelJobs = {
        setImageURL: <Promise<void>[]>[],
        applyAttributes: <Promise<void>[]>[],
        inlineStyles: <Promise<void>[]>[],
    }
    async function processElement(node: CDP.DOM.Node, el: HTMLElement, head: boolean) {
        parallelJobs.setImageURL.push(setImageURL(node, el))
        parallelJobs.applyAttributes.push(applyAttributes(node, el))
        parallelJobs.inlineStyles.push(inlineStyles(node, el))
        for (let child of node.children || []) {
            switch (child.nodeType) {
                case 1: // ELEMENT_NODE
                    if (shouldDropElement(child, head)) continue
                    const childEl = document.createElement(child.nodeName)
                    await processElement(child, childEl, head)
                    el.appendChild(childEl)
                    break
                case 3: // TEXT_NODE
                    const text = document.createTextNode(child.nodeValue)
                    el.appendChild(text)
                    break
                case 4: // CDATA_SECTION_NODE
                    const cdata = document.createCDATASection(child.nodeValue)
                    el.appendChild(cdata)
                    break
            }
        }
    }

    const CDPDocument: CDP.DOM.GetDocumentResponse = await client.send('DOM.getDocument',
        { depth: -1 } as CDP.DOM.GetDocumentRequest)
    // See https://dom.spec.whatwg.org/#interface-node
    for (let node of CDPDocument.root.children!) {
        switch (node.nodeType) {
            case 1: // ELEMENT_NODE
                for (let ch of node.children!) {
                    switch (ch.nodeName) {
                        case "HEAD":
                            await processElement(ch, document.head, true)
                            break
                        case "BODY":
                            await processElement(ch, document.body, false)
                            break
                    }
                }
                break
            case 10: // DOCUMENT_TYPE_NODE
                document.insertBefore(document.implementation.createDocumentType(
                    node.nodeName, node.publicId!, node.systemId!), document.childNodes[0])
                break
        }
    }

    console.timeEnd("sync processing")

    await Promise.all(Object.entries(parallelJobs).map(async ([key, jobs]) => {
        console.time(key)
        await Promise.all(jobs)
        console.timeEnd(key)
    }))

    process.stdout.write(dom.serialize())
}
