const pageLoadTimeout = 60 * 1000 // milliseconds

import * as puppeteer from 'puppeteer'
import * as minimist from 'minimist'

const argv = minimist(process.argv.slice(2), { stopEarly: true });
if (argv._.length != 1) {
    console.error("Missing URL argument")
    process.exit(1)
}
const URL = argv._[0];

function runWithBrowser(f: (browser: puppeteer.Browser) => Promise<void>) {
    (async () => {
        const browser = await puppeteer.launch();
        await f(browser).catch((reason) => { console.error(reason) })
        await browser.close()
    })().catch((reason) => { console.error(reason) })
}

runWithBrowser(async (browser: puppeteer.Browser) => {
    const page = await browser.newPage()
    await page.goto(URL, {
        timeout: pageLoadTimeout,
        waitUntil: ["load", "networkidle0"],
        // https://github.com/GoogleChrome/puppeteer/issues/1353#issuecomment-356561654
    })

    page.on('console', msg => console.warn(msg.text()));

    await page.evaluate(() => {
        function eachElement(document: Document, f: (e: Element) => void) {
            var nodeIterator = document.createNodeIterator(
                document.body,
                NodeFilter.SHOW_ELEMENT
            );

            var node = nodeIterator.nextNode()
            while (node) {
                f(node as Element)
                node = nodeIterator.nextNode()
            }
        }

        eachElement(document, (e: Element) => {
            console.log(e.nodeName)
        })
    })

    process.stdout.write(await page.content())
})


function elementMatchCSSRule(element: Element, cssRule: CSSRule): boolean {
    if (cssRule.type != CSSRule.STYLE_RULE) { return false }
    return element.matches((cssRule as CSSStyleRule).selectorText)
}

var allCSSRules = Array.from(document.styleSheets).reduce(function (rules, styleSheet) {
    if (styleSheet instanceof CSSStyleSheet) {
        return rules.concat(Array.from(styleSheet.cssRules))
    } else {
        return rules
    }
}, [] as CSSRule[])

function getAppliedCSS(e: HTMLElement): CSSRule[] {
    var rules = allCSSRules.filter(elementMatchCSSRule.bind(null, e))
    if (e.style.length > 0) rules.push(e.style.parentRule)

    // TODO: use the CSS property names to lookup the computed values
    // or maybe rewrite in terms of getDefaultComputedStyle diff

    return rules
}
