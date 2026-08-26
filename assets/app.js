/* 1Panel.edu KB — client-side runtime
   Loaded by every module page + index.html.
   Handles: Mermaid render, scroll-spy TOC, anchor links, code copy. */

(function () {
  'use strict';

  // === Mermaid: render <pre class="mermaid"> blocks ===
  function initMermaid() {
    if (typeof mermaid === 'undefined') return;
    mermaid.initialize({
      startOnLoad: true,
      theme: 'dark',
      securityLevel: 'loose',
      themeVariables: {
        primaryColor: '#1e293b',
        primaryTextColor: '#e2e8f0',
        primaryBorderColor: '#38bdf8',
        lineColor: '#94a3b8',
        secondaryColor: '#334155',
        tertiaryColor: '#0f172a',
        fontFamily: 'Inter, system-ui, sans-serif',
      },
      flowchart: { useMaxWidth: true, htmlLabels: true, curve: 'basis' },
      sequence: { useMaxWidth: true, showSequenceNumbers: false },
      er: { useMaxWidth: true },
    });
  }

  // === Anchor links: inject #icon into every h1-h3 ===
  function injectAnchors() {
    const headings = document.querySelectorAll('article h1, article h2, article h3');
    headings.forEach(function (h) {
      if (!h.id) {
        h.id = h.textContent.trim()
          .toLowerCase()
          .replace(/[^\w\u4e00-\u9fff]+/g, '-')
          .replace(/^-+|-+$/g, '')
          .substring(0, 64);
      }
      h.addEventListener('click', function () {
        const url = window.location.origin + window.location.pathname + '#' + h.id;
        navigator.clipboard && navigator.clipboard.writeText(url);
        history.replaceState(null, '', '#' + h.id);
      });
      h.style.cursor = 'pointer';
      h.title = '点击复制锚点链接';
    });
  }

  // === TOC scroll-spy: highlight current section in sidebar ===
  function initScrollSpy() {
    const tocLinks = document.querySelectorAll('aside.toc a[href^="#"]');
    if (tocLinks.length === 0) return;
    const map = new Map();
    tocLinks.forEach(function (a) {
      const id = a.getAttribute('href').substring(1);
      const target = document.getElementById(id);
      if (target) map.set(id, a);
    });
    if (map.size === 0) return;

    const observer = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry) {
        if (entry.isIntersecting) {
          tocLinks.forEach(function (a) { a.classList.remove('active'); });
          const link = map.get(entry.target.id);
          if (link) link.classList.add('active');
        }
      });
    }, { rootMargin: '-80px 0px -70% 0px' });

    map.forEach(function (_, target) {
      const el = document.getElementById(target);
      if (el) observer.observe(el);
    });
  }

  // === Code copy: hover button on every <pre> ===
  function injectCodeCopy() {
    const pres = document.querySelectorAll('article pre');
    pres.forEach(function (pre) {
      if (pre.querySelector('button.copy-btn')) return;
      // detect language from class
      const code = pre.querySelector('code');
      if (code && code.className) {
        const lang = code.className.match(/language-(\w+)/);
        if (lang) pre.setAttribute('data-lang', lang[1]);
      }
      const btn = document.createElement('button');
      btn.className = 'copy-btn';
      btn.textContent = 'copy';
      btn.style.cssText = 'position:absolute;top:6px;right:6px;background:#334155;color:#94a3b8;border:1px solid #475569;border-radius:4px;padding:2px 8px;font-size:11px;cursor:pointer;font-family:inherit;';
      btn.addEventListener('click', function () {
        const txt = pre.innerText;
        navigator.clipboard && navigator.clipboard.writeText(txt);
        btn.textContent = 'copied';
        setTimeout(function () { btn.textContent = 'copy'; }, 1500);
      });
      pre.style.position = 'relative';
      pre.appendChild(btn);
    });
  }

  // === Generate TOC from article headings ===
  function generateTOC() {
    const article = document.querySelector('article');
    const toc = document.querySelector('aside.toc ul');
    if (!article || !toc) return;
    const headings = article.querySelectorAll('h2, h3');
    if (headings.length === 0) return;
    headings.forEach(function (h) {
      if (!h.id) return;
      const li = document.createElement('li');
      const a = document.createElement('a');
      a.href = '#' + h.id;
      a.textContent = h.textContent;
      if (h.tagName === 'H3') a.style.paddingLeft = '12px';
      li.appendChild(a);
      toc.appendChild(li);
    });
  }

  // === Init ===
  function init() {
    injectAnchors();
    generateTOC();
    injectCodeCopy();
    initScrollSpy();
    initMermaid();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
