# gpx-scrubber

A single-page GPX time scrubber. Upload a `.gpx` track, drag the screen-wide
slider to roughly the spot you care about, then use the arrow keys to step one
track point at a time until the marker is exactly where you were — and read off
the precise timestamp.

It's a single self-contained `index.html` (everything runs in your browser, the
file never leaves your machine). It uses [Leaflet](https://leafletjs.com/) and
OpenStreetMap tiles for the map.

**Use it right away:**
[htmlpreview.github.io](https://htmlpreview.github.io/?https://github.com/FiloSottile/mostly-harmless/blob/main/gpx-scrubber/index.html)

## Controls

- **Slider** — coarse scrubbing across the whole track.
- <kbd>←</kbd> / <kbd>→</kbd> — step one point at a time (fine control).
- <kbd>Shift</kbd>+arrow or <kbd>PgUp</kbd> / <kbd>PgDn</kbd> — jump 10 points.
- <kbd>Home</kbd> / <kbd>End</kbd> — jump to the start or end.
- **Click the track** on the map to jump to the nearest point.

For the current point it shows the local and UTC time, elapsed time from the
start, latitude/longitude, elevation, cumulative distance, and instantaneous
speed.
