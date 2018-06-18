const pageLoadTimeout = 60 * 1000 // milliseconds

import * as puppeteer from 'puppeteer'
import * as minimist from 'minimist'

// https://github.com/ChromeDevTools/devtools-protocol/issues/106
import { Protocol as CDP } from 'devtools-protocol'

const argv = minimist(process.argv.slice(2), { stopEarly: true })
if (argv._.length != 1) {
    console.error("Missing URL argument")
    process.exit(1)
}
const URL = argv._[0]

console.warn("Starting...");
(async () => {
    const browser = await puppeteer.launch()
    await runWithContext(browser, async (page: puppeteer.Page, client: puppeteer.CDPSession) => {
        await freezePage(page, client, URL)
    }).catch((reason) => { console.error(reason) })
    await browser.close()
})().catch((reason) => { console.error(reason) })

async function runWithContext(browser: puppeteer.Browser, f: (page: puppeteer.Page, client: puppeteer.CDPSession) => Promise<void>) {
    // https://github.com/DefinitelyTyped/DefinitelyTyped/issues/26626
    const context: puppeteer.Browser = await (browser as any).createIncognitoBrowserContext()
    const page = await context.newPage()
    page.on('console', msg => console.warn(msg.text()))
    const client = await page.target().createCDPSession()
    await client.send('DOM.enable')
    await client.send('CSS.enable')
    await f(page, client).catch((reason) => { console.error(reason) })
    await client.detach()
    await context.close()
}

async function freezePage(page: puppeteer.Page, client: puppeteer.CDPSession, URL: string) {
    console.warn("Fetching page " + URL + "...")
    await page.setViewport({ width: 1280, height: 850 })
    await page.goto(URL, {
        timeout: pageLoadTimeout,
        waitUntil: ["load", "networkidle0"],
        // https://github.com/GoogleChrome/puppeteer/issues/1353#issuecomment-356561654
    })
    console.warn("... fetched page")

    async function getMatchedStyle(nodeId: CDP.DOM.NodeId): Promise<CDP.CSS.GetMatchedStylesForNodeResponse> {
        return await client.send('CSS.getMatchedStylesForNode',
            { nodeId: nodeId } as CDP.CSS.GetMatchedStylesForNodeRequest)
    }
    async function getComputedStyle(nodeId: CDP.DOM.NodeId): Promise<Map<string, string>> {
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

    async function inlineStyles(nodeId: CDP.DOM.NodeId): Promise<void> {
        const computedStyle = await getComputedStyle(nodeId)
        const styledProps = await getStyledProperties(nodeId)

        var style = ""
        for (var name of styledProps) {
            if (!computedStyle.get(name)) continue
            style += name + ":" + computedStyle.get(name) + ";"
        }
        style = style.replace(/"/g, "'") // TODO: almost certainly broken
        if (style != "") {
            await client.send('DOM.setAttributeValue', {
                nodeId: nodeId, name: "style", value: style
            } as CDP.DOM.SetAttributeValueRequest)
        }
    }

    const CDPDocument: CDP.DOM.GetFlattenedDocumentResponse = await client.send('DOM.getFlattenedDocument',
        { depth: -1 } as CDP.DOM.GetFlattenedDocumentRequest)
    for (var node of CDPDocument.nodes) {
        if (node.nodeType != 1 /* ELEMENT_NODE */) continue // https://dom.spec.whatwg.org/#dom-node-nodetype
        await inlineStyles(node.nodeId)
    }

    console.warn("... inlined styles")

    await page.evaluate(() => {
        function eachElement(document: Document, f: (e: Element) => void) {
            var nodeIterator = document.createNodeIterator(document, NodeFilter.SHOW_ELEMENT)
            var node = nodeIterator.nextNode()
            while (node) {
                f(node as Element)
                node = nodeIterator.nextNode()
            }
        }

        eachElement(document, (e: Element) => {
            // TODO: proper element and attribute whitelist
            if (e.nodeName == "SCRIPT" || e.nodeName == "STYLE") {
                e.remove()
                return
            }
            if (e.nodeName == "LINK" && (e.getAttribute("rel") || "").toLowerCase() == "stylesheet") {
                e.remove()
                return
            }
            if (e.nodeName == "META" && e.getAttribute("http-equiv")) {
                e.remove()
                return
            }
            if (document.head.contains(e)) return
            for (var attr of Array.from(e.attributes)) {
                if (attr.name != "style" && attr.name != "href" && attr.name != "datetime") {
                    e.removeAttributeNode(attr)
                }
            }
        })
    })

    console.warn("... stripped DOM")

    process.stdout.write(await page.content())
}
