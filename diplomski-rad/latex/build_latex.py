#!/usr/bin/env python3
"""Convert part1..3.md (the Markdown source of the thesis) into LaTeX chapter
files under chapters/, reusing the same source text as the Markdown/PDF
deliverable but targeting LaTeX output (\\chapter/\\section, longtable,
lstlisting, \\cite{}, figure environments for the diagrams).

Front matter (cover/KDI/Zadatak/Izjava), the literature chapter (handled by
biblatex/\\printbibliography), Spisak skracenica/slika/tabela and Biografija
are hand-written directly in chapters/00-frontmatter.tex, 12-spiskovi.tex and
13-biografija.tex - this script only emits the numbered content chapters
1-10, plus references.bib is maintained by hand (see references.bib).
"""
import html
import os
import re

os.chdir(os.path.dirname(os.path.abspath(__file__)))

REFS_KEYS = {
    "TANENBAUM", "DIJKSTRA", "ASTAR", "VRP", "OSMWIKI", "WEBSOCKET", "JWT",
    "GEOFABRIK", "OSMIUM", "AETR", "GO", "FLUTTER", "VALHALLA", "POSTGIS",
    "RABBITMQ", "DOCKER", "NOMINATIM", "GOOSE",
}

CHAPTER_FILES = {
    1: "01-uvod", 2: "02-pregled", 3: "03-tehnologije", 4: "04-arhitektura",
    5: "05-osm", 6: "06-algoritam", 7: "07-backend", 8: "08-mobilna",
    9: "09-testiranje", 10: "10-zakljucak",
}

# ---------------------------------------------------------------- slicing ---
p1 = open("../part1.md", encoding="utf-8").read()
p2 = open("../part2.md", encoding="utf-8").read()
p3 = open("../part3.md", encoding="utf-8").read()

p1_body = p1[p1.index("# 1 Uvod"):]
p3_body = p3[p3.index("# 8 Mobilna aplikacija"): p3.index("# Literatura")]

text = "\n\n".join([p1_body, p2, p3_body])

# --------------------------------------------------------- citation marks --
CITE_OPEN, CITE_CLOSE = "\x02", "\x03"


def mark_cite(m):
    k = m.group(1)
    return f"{CITE_OPEN}{k}{CITE_CLOSE}" if k in REFS_KEYS else m.group(0)


text = re.sub(r"\[([A-Z][A-Z0-9]*)\]", mark_cite, text)

# -------------------------------------------------------- stash code fences #
CODE = []


def stash_code(m):
    CODE.append(m.group(2))
    return f"\x00CODE{len(CODE) - 1}\x00"


text = re.sub(r"```(\w*)\n(.*?)```", stash_code, text, flags=re.S)

# ------------------------------------------------------------- LaTeX escape #
_ESCAPE_MAP = [
    ("\\", r"\textbackslash{}"),
    ("&", r"\&"),
    ("%", r"\%"),
    ("$", r"\$"),
    ("#", r"\#"),
    ("_", r"\_"),
    ("{", r"\{"),
    ("}", r"\}"),
    ("~", r"\textasciitilde{}"),
    ("^", r"\textasciicircum{}"),
]


def esc(s):
    for a, b in _ESCAPE_MAP:
        s = s.replace(a, b)
    return s


def inline(s):
    stash = []

    def put(rep):
        stash.append(rep)
        return f"\x04{len(stash) - 1}\x05"

    s = re.sub(rf"{CITE_OPEN}([A-Z][A-Z0-9]*){CITE_CLOSE}",
                lambda m: put(r"\cite{%s}" % m.group(1).lower()), s)
    s = re.sub(r"\$(.+?)\$", lambda m: put("$" + m.group(1) + "$"), s)
    s = re.sub(r"`([^`]+?)`", lambda m: put(r"\texttt{%s}" % esc(m.group(1))), s)

    s = esc(s)
    s = re.sub(r"\*\*(.+?)\*\*", r"\\textbf{\1}", s)
    s = re.sub(r"(?<!\*)\*(?!\*)(.+?)(?<!\*)\*(?!\*)", r"\\emph{\1}", s)

    s = re.sub(r"\x04(\d+)\x05", lambda m: stash[int(m.group(1))], s)
    return s


CAPTION_RE = re.compile(r"^\*\*(Slika|Tabela|Listing)\s+(\d+)\.(\d+)\.\*\*\s*(.*)$")


def short_caption(caption_latex):
    """Kratak naslov listinga za "Spisak listinga".

    Pun naslov cesto sadrzi putanju do fajla u zagradi i objasnjenje posle
    crte; u spisku listinga takav red je predugacak (putanje se ne mogu
    prelomiti) pa izlazi van margine. Zato se za spisak koristi skraceni
    oblik: bez zagrade sa putanjom i bez dela posle " --- "/" - ".
    """
    s = re.sub(r"\s*\(\\texttt\{[^{}]*\}\)", "", caption_latex)
    s = re.split(r"\s+[—–-]{1,3}\s+", s)[0]
    return s.strip().rstrip(",;:")


def render_table(lines, caption_latex=None, label=None):
    rows = [ln.strip().strip("|").split("|") for ln in lines]
    rows = [[c.strip() for c in r] for r in rows]
    header, sep, *body = rows
    n = len(header)
    colw = 0.94 / n
    colspec = ("p{%.3f\\textwidth}" % colw) * n
    out = [r"\begin{longtable}{@{}" + colspec + "@{}}"]
    if caption_latex:
        out.append(r"\caption{%s}\label{%s}\\" % (caption_latex, label))
    out.append(r"\toprule")
    out.append(" & ".join(r"\textbf{%s}" % inline(c) for c in header) + r" \\")
    out.append(r"\midrule")
    out.append(r"\endhead")
    for r in body:
        out.append(" & ".join(inline(c) for c in r) + r" \\")
    out.append(r"\bottomrule")
    out.append(r"\end{longtable}")
    return "\n".join(out)


def render_list(lines):
    out = []
    stack = []
    depth = 0
    for ln in lines:
        indent = len(ln) - len(ln.lstrip(" "))
        content = ln.strip()
        ordered = bool(re.match(r"^\d+\.\s", content))
        item_text = re.sub(r"^(-|\d+\.)\s", "", content)
        target_depth = 1 if indent >= 2 else 0
        while depth < target_depth + 1:
            env = "enumerate" if ordered else "itemize"
            out.append(f"\\begin{{{env}}}")
            stack.append(env)
            depth += 1
        while depth > target_depth + 1:
            out.append(f"\\end{{{stack.pop()}}}")
            depth -= 1
        out.append(r"\item " + inline(item_text))
    while stack:
        out.append(f"\\end{{{stack.pop()}}}")
    return "\n".join(out)


blocks = re.split(r"\n\s*\n", text)
blocks = [b for b in blocks if b.strip()]

DIAGRAM_RE = re.compile(r"^\[DIAGRAM:(\w+)\]$")
CODE_PH_RE = re.compile(r"^\x00CODE(\d+)\x00$")

chapter_blocks = []  # list of (chapter_num, latex_str)
current_chapter = 1
i = 0
while i < len(blocks):
    block = blocks[i].strip("\n")
    lines = block.split("\n")
    first = lines[0]
    consumed = 1
    rendered = None

    m_diag = DIAGRAM_RE.match(first)
    m_code = CODE_PH_RE.match(first)
    m_tabcap = CAPTION_RE.match(first) if len(lines) == 1 else None

    if m_diag and i + 1 < len(blocks) and CAPTION_RE.match(blocks[i + 1].strip()):
        cap = CAPTION_RE.match(blocks[i + 1].strip())
        name = m_diag.group(1)
        rendered = (
            "\\begin{figure}[h]\n\\centering\n"
            f"\\includegraphics[width=0.88\\textwidth]{{diagrams/{name}.pdf}}\n"
            f"\\caption{{{inline(cap.group(4))}}}\n"
            f"\\label{{fig:{name}}}\n"
            "\\end{figure}"
        )
        consumed = 2
    elif m_code and i + 1 < len(blocks) and CAPTION_RE.match(blocks[i + 1].strip()) \
            and CAPTION_RE.match(blocks[i + 1].strip()).group(1) == "Listing":
        cap = CAPTION_RE.match(blocks[i + 1].strip())
        code = CODE[int(m_code.group(1))]
        if code.endswith("\n"):
            code = code[:-1]
        full_cap = inline(cap.group(4))
        short_cap = short_caption(full_cap)
        cap_opt = full_cap if short_cap == full_cap else "[%s]%s" % (short_cap, full_cap)
        rendered = (
            "\\begin{lstlisting}[caption={%s}]\n%s\n\\end{lstlisting}"
            % (cap_opt, code)
        )
        consumed = 2
    elif m_tabcap and m_tabcap.group(1) == "Tabela" and i + 1 < len(blocks) \
            and all(l.strip().startswith("|") for l in blocks[i + 1].strip().split("\n")):
        n1, n2 = m_tabcap.group(2), m_tabcap.group(3)
        table_lines = [l for l in blocks[i + 1].strip().split("\n") if l.strip()]
        rendered = render_table(table_lines, inline(m_tabcap.group(4)), f"tab:{n1}-{n2}")
        consumed = 2
    elif block.strip() == "<!-- PAGEBREAK -->":
        rendered = "\\clearpage"
    elif re.match(r"^#{1,2}\s", first):
        level = len(re.match(r"^(#{1,2})", first).group(1))
        content = first.lstrip("#").strip()
        if level == 1:
            mnum = re.match(r"^(\d+)\s+(.*)$", content)
            current_chapter = int(mnum.group(1))
            rendered = "\\chapter{%s}" % inline(mnum.group(2))
        else:
            mnum = re.match(r"^(\d+\.\d+)\s+(.*)$", content)
            rendered = "\\section{%s}" % inline(mnum.group(2))
    elif all(l.strip().startswith("|") for l in lines):
        rendered = render_table(lines)
    elif all(re.match(r"^\s*(-\s|\d+\.\s)", l) for l in lines if l.strip()):
        rendered = render_list([l for l in lines if l.strip()])
    else:
        rendered = inline(" ".join(l.strip() for l in lines))

    chapter_blocks.append((current_chapter, rendered))
    i += consumed

# ------------------------------------------------------------------- write --
by_chapter = {}
for num, block in chapter_blocks:
    by_chapter.setdefault(num, []).append(block)

for num, slug in CHAPTER_FILES.items():
    content = "\n\n".join(by_chapter.get(num, []))
    with open(f"chapters/{slug}.tex", "w", encoding="utf-8") as f:
        f.write(content + "\n")

print("Wrote", ", ".join(f"chapters/{s}.tex" for s in CHAPTER_FILES.values()))
