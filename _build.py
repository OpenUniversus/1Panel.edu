"""Convert all modules/XX-xxx/HUMAN-READABLE.md → HUMAN-READABLE.html (HTML+CSS+JS).

No external deps. Pure Python markdown parser supporting:
  - ATX headings (# / ## / ### ...)
  - Paragraphs
  - **bold** *italic* `inline code` [text](url)
  - Unordered (- ) and ordered (1. ) lists
  - Fenced code blocks (``` ... ```) with language tag
  - Mermaid code blocks (```mermaid ... ```)
  - Tables (| col | col |)
  - Blockquotes (> ...)
  - Horizontal rules (---)
  - HTML pass-through (raw <iframe>, <details>, etc.)

Builds with unified assets/style.css + assets/app.js.
"""
import os
import re
import shutil
import sys
from pathlib import Path

ROOT = Path(r"D:\MiniMax Code\1Panel\1Panel.edu")
MODULES = ROOT / "modules"
ASSETS = ROOT / "assets"

HTML_TEMPLATE = """<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{title} · 1Panel.edu KB</title>
  <link rel="stylesheet" href="{style_path}">
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Noto+Serif+SC:wght@400;600&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
  <script src="https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.min.js"></script>
</head>
<body>
  <nav class="top">
    <a class="brand" href="{root}index.html"><span class="logo">1P</span>1Panel.edu KB</a>
    <div class="nav-links">
      <a href="{root}index.html">目录</a>
      <a href="{root}modules/{rel_path}#">本模块</a>
      <a href="https://github.com/OpenUniversus/1Panel.edu" target="_blank" rel="noopener">GitHub ↗</a>
    </div>
  </nav>
  <div class="layout">
    <aside class="toc">
      <h3>本文目录</h3>
      <ul></ul>
    </aside>
    <main>
      <header class="doc">
        <h1>{title}</h1>
        <div class="meta">
          <span class="pill {pill_class}">{state}</span>
          <span>{priority}</span>
          <span>{size_kb} KB</span>
          <span>{updated}</span>
        </div>
      </header>
      <article>
{body}
      </article>
    </main>
  </div>
  <footer>
    <p>1Panel.edu KB · <a href="{root}index.html">返回目录</a> · <a href="https://github.com/OpenUniversus/1Panel.edu" target="_blank" rel="noopener">GitHub</a></p>
  </footer>
  <script src="{app_path}"></script>
</body>
</html>
"""


def escape_html(text):
    return text.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def inline(text):
    """Apply inline markdown: code, bold, italic, link."""
    # code spans (don't touch content inside)
    text = re.sub(r"`([^`]+)`", r"<code>\1</code>", text)
    # links
    text = re.sub(r"\[([^\]]+)\]\(([^)]+)\)", r'<a href="\2">\1</a>', text)
    # bold
    text = re.sub(r"\*\*([^*]+)\*\*", r"<strong>\1</strong>", text)
    # italic
    text = re.sub(r"(?<!\*)\*([^*]+)\*(?!\*)", r"<em>\1</em>", text)
    return text


def render_table(lines, start_idx):
    """Render a markdown table. Returns (html, end_idx)."""
    rows = []
    i = start_idx
    while i < len(lines) and lines[i].strip().startswith("|"):
        rows.append([c.strip() for c in lines[i].strip().strip("|").split("|")])
        i += 1
    if len(rows) < 2:
        return "", start_idx
    # row 1 is header, row 2 is alignment (skip)
    header = rows[0]
    body = rows[2:]
    html = "<table><thead><tr>"
    for h in header:
        html += f"<th>{inline(escape_html(h))}</th>"
    html += "</tr></thead><tbody>"
    for row in body:
        html += "<tr>"
        for cell in row:
            html += f"<td>{inline(escape_html(cell))}</td>"
        html += "</tr>"
    html += "</tbody></table>"
    return html, i


def render_code_block(lines, start_idx):
    """Render fenced code block. Returns (html, end_idx, language)."""
    first = lines[start_idx]
    lang_match = re.match(r"^```(\w*)\s*$", first)
    lang = lang_match.group(1) if lang_match else ""
    body = []
    i = start_idx + 1
    while i < len(lines) and not lines[i].rstrip().startswith("```"):
        body.append(lines[i])
        i += 1
    code = "\n".join(body)
    cls = f' class="language-{lang}"' if lang else ""
    if lang == "mermaid":
        html = f'<pre class="mermaid">{escape_html(code)}</pre>'
    else:
        html = f'<pre data-lang="{lang}"><code{cls}>{escape_html(code)}</code></pre>'
    return html, i + 1


def md_to_html(md_text):
    """Convert markdown text to HTML body fragment."""
    lines = md_text.split("\n")
    out = []
    i = 0
    in_list = None  # None | 'ul' | 'ol'
    in_para = []

    def flush_para():
        nonlocal in_para
        if in_para:
            text = " ".join(in_para)
            out.append(f"<p>{inline(escape_html(text))}</p>")
            in_para = []

    def flush_list():
        nonlocal in_list
        if in_list:
            out.append(f"</{in_list}>")
            in_list = None

    while i < len(lines):
        line = lines[i]
        stripped = line.strip()

        # code block
        if stripped.startswith("```"):
            flush_para(); flush_list()
            html, i = render_code_block(lines, i)
            out.append(html)
            continue

        # heading
        m = re.match(r"^(#{1,6})\s+(.+)$", stripped)
        if m:
            flush_para(); flush_list()
            level = len(m.group(1))
            out.append(f"<h{level}>{inline(escape_html(m.group(2)))}</h{level}>")
            i += 1
            continue

        # table
        if stripped.startswith("|"):
            flush_para(); flush_list()
            html, i = render_table(lines, i)
            if html:
                out.append(html)
                continue

        # horizontal rule
        if stripped == "---":
            flush_para(); flush_list()
            out.append("<hr>")
            i += 1
            continue

        # blockquote
        if stripped.startswith(">"):
            flush_para(); flush_list()
            bq = []
            while i < len(lines) and lines[i].strip().startswith(">"):
                bq.append(lines[i].strip().lstrip(">").strip())
                i += 1
            text = " ".join(bq)
            out.append(f"<blockquote>{inline(escape_html(text))}</blockquote>")
            continue

        # list item
        ul = re.match(r"^[-*]\s+(.+)$", stripped)
        ol = re.match(r"^\d+\.\s+(.+)$", stripped)
        if ul or ol:
            flush_para()
            new_type = "ul" if ul else "ol"
            if in_list != new_type:
                flush_list()
                in_list = new_type
                out.append(f"<{new_type}>")
            content = (ul or ol).group(1)
            out.append(f"<li>{inline(escape_html(content))}</li>")
            i += 1
            continue

        # blank line
        if not stripped:
            flush_para(); flush_list()
            i += 1
            continue

        # paragraph (collect)
        in_para.append(stripped)
        i += 1

    flush_para()
    flush_list()
    return "\n".join(out)


def find_module_meta(module_dir):
    """Extract title, priority, state from a module's name/dir or default."""
    # Use the parent dir name as a hint, e.g. "14-auth" -> "14. Auth 登录认证"
    name = module_dir.name
    mapping = {
        "14-auth": ("Auth 登录认证", "⭐⭐⭐", "done", "DONE v1"),
        "01-app-store": ("App 应用商店", "⭐⭐⭐", "done", "DONE v2"),
        "03-website": ("Website 网站管理", "⭐⭐⭐", "done", "DONE v3"),
        "02-container": ("Docker 容器管理", "⭐⭐⭐", "frozen", "FROZEN"),
        "12-security": ("Nginx (12-security 拆)", "⭐⭐", "todo", "TODO"),
        "12-ssl": ("SSL 证书 (12-security 拆)", "⭐⭐", "todo", "TODO"),
        "06-cronjob": ("CronJob 定时任务", "⭐⭐", "todo", "TODO"),
        "08-file": ("File 文件管理", "⭐⭐", "todo", "TODO"),
        "10-host-monitor": ("Monitor 监控", "⭐", "todo", "TODO"),
        "05-backup-snapshot": ("Backup 备份", "⭐", "todo", "TODO"),
        "15-settings": ("Settings 系统设置 (新)", "⭐", "todo", "TODO (新)"),
        "16-terminal": ("Terminal WebSSH (新)", "⭐", "todo", "TODO (新)"),
        "04-database": ("Database 数据库", "-", "todo", "TODO"),
        "07-alert": ("Alert 告警", "-", "todo", "TODO"),
        "09-ai-agent": ("AI Agent", "-", "todo", "TODO"),
        "11-runtime-ai": ("Runtime AI", "-", "todo", "TODO"),
        "13-frontend": ("Frontend 前端 (Vue 3)", "-", "todo", "TODO"),
    }
    return mapping.get(name, (name, "?", "todo", "TODO"))


def build_module(module_dir):
    """Build one module's HR.md → HR.html."""
    hr_md = module_dir / "HUMAN-READABLE.md"
    if not hr_md.exists():
        return False, f"no HUMAN-READABLE.md"

    title, priority, state, state_label = find_module_meta(module_dir)
    md_text = hr_md.read_text(encoding="utf-8", errors="ignore")
    # strip the first H1 (we provide it in template)
    md_text = re.sub(r"^#\s+.+?\n", "", md_text, count=1, flags=re.MULTILINE)
    body = md_to_html(md_text)

    # file size
    size = hr_md.stat().st_size
    size_kb = round(size / 1024, 1)
    import datetime
    updated = datetime.datetime.fromtimestamp(hr_md.stat().st_mtime).strftime("%Y-%m-%d")

    # path back to root
    rel_depth = len(module_dir.relative_to(ROOT).parts)  # e.g. 2 for modules/14-auth
    root = "../" * rel_depth
    style_path = f"{root}assets/style.css"
    app_path = f"{root}assets/app.js"

    rel_path = module_dir.relative_to(ROOT).as_posix()

    html = HTML_TEMPLATE.format(
        title=escape_html(title),
        pill_class=f"pill-{state}",
        state=state_label,
        priority=priority,
        size_kb=size_kb,
        updated=updated,
        body=body,
        root=root,
        style_path=style_path,
        app_path=app_path,
        rel_path=rel_path,
    )

    out_path = hr_md.with_suffix(".html")
    out_path.write_text(html, encoding="utf-8")
    return True, f"{hr_md.name} → {out_path.name} ({size_kb} KB)"


def main():
    print("=== Build HR.md → HR.html ===\n")
    ok = 0
    skip = 0
    for module_dir in sorted(MODULES.iterdir()):
        if not module_dir.is_dir():
            continue
        success, msg = build_module(module_dir)
        if success:
            print(f"  OK  {msg}")
            ok += 1
        else:
            print(f"  --  {module_dir.name}: {msg}")
            skip += 1
    print(f"\n{ok} modules built, {skip} skipped")


if __name__ == "__main__":
    main()
