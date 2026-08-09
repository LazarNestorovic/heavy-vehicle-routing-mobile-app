#!/usr/bin/env python3
"""Assemble part1..3.md into Diplomski_rad.md (citations numbered, literature
generated, diagrams/pagebreaks injected) and render Diplomski_rad.html."""
import html
import re

PARTS = ["part1.md", "part2.md", "part3.md"]

REFS = {
    "TANENBAUM": "A. S. Tanenbaum, M. van Steen, <i>Distributed Systems: Principles and Paradigms</i>, 3. izdanje, Pearson, 2017.",
    "DIJKSTRA": "E. W. Dijkstra, „A note on two problems in connexion with graphs“, <i>Numerische Mathematik</i>, vol. 1, str. 269–271, 1959.",
    "ASTAR": "P. E. Hart, N. J. Nilsson, B. Raphael, „A Formal Basis for the Heuristic Determination of Minimum Cost Paths“, <i>IEEE Transactions on Systems Science and Cybernetics</i>, vol. 4, br. 2, str. 100–107, 1968.",
    "VRP": "G. B. Dantzig, J. H. Ramser, „The Truck Dispatching Problem“, <i>Management Science</i>, vol. 6, br. 1, str. 80–91, 1959.",
    "OSMWIKI": "OpenStreetMap Wiki, „Map Features“, https://wiki.openstreetmap.org/wiki/Map_features (pristupljeno avgusta 2026).",
    "WEBSOCKET": "I. Fette, A. Melnikov, „The WebSocket Protocol“, RFC 6455, IETF, 2011.",
    "JWT": "M. Jones, J. Bradley, N. Sakimura, „JSON Web Token (JWT)“, RFC 7519, IETF, 2015.",
    "GEOFABRIK": "Geofabrik GmbH, „OpenStreetMap Data Extracts“, https://download.geofabrik.de (pristupljeno 2026).",
    "OSMIUM": "„osmium-tool — command line tool for working with OpenStreetMap data“, https://osmcode.org/osmium-tool/.",
    "AETR": "Uredba (EZ) br. 561/2006 Evropskog parlamenta i Saveta o usklađivanju određenih socijalnih propisa u vezi sa drumskim saobraćajem (AETR).",
    "GO": "„The Go Programming Language“, Google, https://go.dev.",
    "FLUTTER": "„Flutter — Build apps for any screen“, Google, https://docs.flutter.dev.",
    "VALHALLA": "„Valhalla Routing Engine“, dokumentacija, https://valhalla.github.io/valhalla/.",
    "POSTGIS": "„PostGIS — Spatial and Geographic Objects for PostgreSQL“, https://postgis.net.",
    "RABBITMQ": "„RabbitMQ Documentation“, https://www.rabbitmq.com/documentation.html.",
    "DOCKER": "„Docker Compose overview“, https://docs.docker.com/compose/.",
    "NOMINATIM": "„Nominatim — OpenStreetMap geocoding“, https://nominatim.org.",
    "GOOSE": "„goose — database migration tool“, https://github.com/pressly/goose.",
}

# ---------------------------------------------------------------- assembly --
text = "\n\n".join(open(p, encoding="utf-8").read() for p in PARTS)

order = []
for m in re.finditer(r"\[([A-Z][A-Z0-9]*)\]", text):
    k = m.group(1)
    if k in REFS and k not in order:
        order.append(k)
numbering = {k: i + 1 for i, k in enumerate(order)}


def cite_repl(m):
    k = m.group(1)
    return f"[{numbering[k]}]" if k in numbering else m.group(0)


text = re.sub(r"\[([A-Z][A-Z0-9]*)\]", cite_repl, text)

lit_items = "\n".join(f"<li id=\"ref{i+1}\">{REFS[k]}</li>" for i, k in enumerate(order))
lit_html = f'<ol class="literature">\n{lit_items}\n</ol>'

quote_html = (
    '<div class="quotepage"><blockquote>'
    '„The map is not the territory.“'
    '<footer>— Alfred Korzybski</footer>'
    "</blockquote></div>"
)

svg_architecture = open("diagrams/architecture.svg", encoding="utf-8").read()
svg_explain = open("diagrams/explain.svg", encoding="utf-8").read()
svg_erd = open("diagrams/erd.svg", encoding="utf-8").read()

text = text.replace("<!-- LITERATURE_LIST -->", lit_html)
text = text.replace("<!-- QUOTE_PAGE -->", quote_html)
text = text.replace("[DIAGRAM:architecture]", f'<div class="diagram">{svg_architecture}</div>')
text = text.replace("[DIAGRAM:explain]", f'<div class="diagram">{svg_explain}</div>')
text = text.replace("[DIAGRAM:erd]", f'<div class="diagram">{svg_erd}</div>')

# Chromium's print-to-pdf inserts an extra BLANK page when a forced
# page-break-before lands on an element that (due to upstream content height)
# already starts a fresh page. Attaching the break to the END of the PRECEDING
# block instead (page-break-after on real content, never on an empty element)
# avoids that quirk entirely. PB_MARK has no blank line before it, so it merges
# into the previous block during the blank-line block-split below; the blank
# line AFTER it still starts the next block normally.
PB_MARK = "\x01PAGEBREAK\x01"
text = re.sub(r"\n\s*\n<!-- PAGEBREAK -->\n\s*\n", f"\n{PB_MARK}\n\n", text)

with open("Diplomski_rad.md", "w", encoding="utf-8") as f:
    f.write(text)

# -------------------------------------------------------------- MD -> HTML --
CODE_PLACEHOLDER = []


def stash_code(m):
    lang = m.group(1).strip()
    body = html.escape(m.group(2))
    CODE_PLACEHOLDER.append(f'<pre class="code lang-{lang or "text"}"><code>{body}</code></pre>')
    return f"\x00CODE{len(CODE_PLACEHOLDER) - 1}\x00"


text = re.sub(r"```(\w*)\n(.*?)```", stash_code, text, flags=re.S)


def inline(s):
    s = html.escape(s, quote=False)

    def math_repl(m):
        content = m.group(1).replace("\\log", "log").replace("\\cdot", "·")
        return f"<em>{content}</em>"

    s = re.sub(r"\$(.+?)\$", math_repl, s)
    s = re.sub(r"`([^`]+?)`", r"<code>\1</code>", s)
    s = re.sub(r"\*\*(.+?)\*\*", r"<strong>\1</strong>", s)
    s = re.sub(r"(?<!\*)\*(?!\*)(.+?)(?<!\*)\*(?!\*)", r"<em>\1</em>", s)
    s = re.sub(r"\[(\d+)\]", r'<a class="cite" href="#ref\1">[\1]</a>', s)
    return s


def render_table(lines):
    rows = [ln.strip().strip("|").split("|") for ln in lines]
    rows = [[c.strip() for c in r] for r in rows]
    header, sep, *body = rows
    for r in body:
        if len(r) != len(header):
            raise ValueError(f"table row column mismatch: header={header} row={r}")
    out = ["<table>", "<thead><tr>"]
    for c in header:
        out.append(f"<th>{inline(c)}</th>")
    out.append("</tr></thead><tbody>")
    for r in body:
        out.append("<tr>")
        for c in r:
            out.append(f"<td>{inline(c)}</td>")
        out.append("</tr>")
    out.append("</tbody></table>")
    return "\n".join(out)


def render_list(lines):
    # supports one level of indentation (2+ leading spaces = nested)
    html_out = []
    stack_ordered = []
    depth = 0
    for ln in lines:
        indent = len(ln) - len(ln.lstrip(" "))
        content = ln.strip()
        ordered = bool(re.match(r"^\d+\.\s", content))
        item_text = re.sub(r"^(-|\d+\.)\s", "", content)
        target_depth = 1 if indent >= 2 else 0
        while depth < target_depth + 1:
            tag = "ol" if ordered else "ul"
            html_out.append(f"<{tag}>")
            stack_ordered.append(tag)
            depth += 1
        while depth > target_depth + 1:
            html_out.append(f"</{stack_ordered.pop()}>")
            depth -= 1
        html_out.append(f"<li>{inline(item_text)}</li>")
    while stack_ordered:
        html_out.append(f"</{stack_ordered.pop()}>")
    return "\n".join(html_out)


blocks = re.split(r"\n\s*\n", text)
html_blocks = []
for block in blocks:
    block = block.strip("\n")
    if not block.strip():
        continue

    pb_after = False
    stripped = block.rstrip()
    if stripped.endswith(PB_MARK):
        pb_after = True
        block = stripped[: -len(PB_MARK)].rstrip("\n")

    lines = block.split("\n")
    first = lines[0]
    rendered = None

    if re.match(r"^\x00CODE\d+\x00$", first) and len(lines) == 1:
        rendered = first
    elif first.lstrip().startswith("<"):
        rendered = block
    elif all(l.strip().startswith("|") for l in lines):
        rendered = render_table(lines)
    elif re.match(r"^#{1,3}\s", first):
        level = len(re.match(r"^(#{1,3})", first).group(1))
        content = first.lstrip("#").strip()
        parts = [f"<h{level}>{inline(content)}</h{level}>"]
        if len(lines) > 1:
            rest = " ".join(l.strip() for l in lines[1:])
            parts.append(f"<p>{inline(rest)}</p>")
        rendered = "\n".join(parts)
    elif all(re.match(r"^\s*(-\s|\d+\.\s)", l) for l in lines if l.strip()):
        rendered = render_list([l for l in lines if l.strip()])
    elif first.strip() == "---":
        rendered = "<hr/>"
    else:
        rendered = f"<p>{inline(' '.join(l.strip() for l in lines))}</p>"

    if pb_after:
        rendered = f'<div style="page-break-after:always;break-after:page;">{rendered}</div>'
    html_blocks.append(rendered)

body_html = "\n".join(html_blocks)
for i, c in enumerate(CODE_PLACEHOLDER):
    body_html = body_html.replace(f"\x00CODE{i}\x00", c)

PAGE_CSS = open("style.css", encoding="utf-8").read()

full_html = f"""<!doctype html>
<html lang="sr">
<head>
<meta charset="utf-8"/>
<title>Diplomski rad</title>
<style>{PAGE_CSS}</style>
</head>
<body>
{body_html}
</body>
</html>
"""

with open("Diplomski_rad.html", "w", encoding="utf-8") as f:
    f.write(full_html)

print(f"Citations: {len(order)} unique references numbered 1..{len(order)}")
print("Wrote Diplomski_rad.md and Diplomski_rad.html")

import subprocess

subprocess.run(
    [
        "google-chrome", "--headless=new", "--disable-gpu", "--no-sandbox",
        "--print-to-pdf=Diplomski_rad.pdf", "--no-pdf-header-footer",
        f"file://{__import__('os').path.abspath('Diplomski_rad.html')}",
    ],
    check=True,
)
print("Wrote Diplomski_rad.pdf")
