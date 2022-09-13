(() => {
    const patterns = [
        [/github\.com\/golang\/go\/issues\/(\d+)/i, "https://go.dev/issue/"],
        [/go-review\.googlesource\.com\/c\/.*\/\+\/([\d\/]+)/i, "https://go.dev/cl/"],
        [/github\.com\/golang\/go\/wiki\/(\w+)/i, "https://go.dev/wiki/"],
        [/go\.googlesource\.com\/proposal\/\+\/master\/design\/([\w\-]+)\.md/i, "https://go.dev/design/"],
    ];
    for (const p of patterns) {
        const match = window.location.href.match(p[0]);
        if (match) {
            navigator.clipboard.writeText(p[1] + match[1]);
            return;
        }
    }
    navigator.clipboard.writeText(window.location.href);
})()
