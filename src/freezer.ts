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
