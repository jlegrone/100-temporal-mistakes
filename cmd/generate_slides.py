#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "python-pptx",
#     "pyyaml",
#     "playwright",
# ]
# ///
"""Generate a PowerPoint presentation from SLIDES.md.

SLIDES.md format:
  - Optional YAML frontmatter (between --- delimiters) for presentation metadata
  - Slides separated by --- on its own line (not indented, not inside code fences)
  - Markdown content: # Title, ## Subtitle, - bullets, **bold**, *italic*
  - Code blocks with language hints: ```go, ```bash, ```diff
  - HTML comments (<!-- ... -->) become speaker notes

Usage:
    uv run cmd/generate_slides.py [SLIDES.md] [-o output.pptx]

First-time setup for carbon screenshots:
    uv run --with playwright python -m playwright install chromium
"""

from __future__ import annotations

import argparse
import asyncio
import hashlib
import re
import sys
from dataclasses import dataclass, field
from pathlib import Path
from urllib.parse import quote, urlencode

import yaml
from pptx import Presentation
from pptx.dml.color import RGBColor
from pptx.enum.text import PP_ALIGN
from pptx.util import Inches, Pt

# ---------------------------------------------------------------------------
# Data model
# ---------------------------------------------------------------------------


@dataclass
class ThemeConfig:
    background_color: str = "#FFFFFF"
    title_color: str = "#1E1E2E"
    text_color: str = "#333333"
    accent_color: str = "#7C3AED"
    code_bg_color: str = "#1E1E2E"
    code_text_color: str = "#D4D4D4"
    diff_add_color: str = "#4EC9B0"
    diff_remove_color: str = "#F44747"
    font_heading: str = "Arial"
    font_body: str = "Arial"
    code_font: str = "Courier New"


@dataclass
class PresentationMeta:
    title: str = "Untitled"
    subtitle: str = ""
    author: str = ""
    theme: ThemeConfig = field(default_factory=ThemeConfig)


@dataclass
class CodeBlock:
    language: str  # "go", "bash", "diff", ""
    code: str


@dataclass
class ImageRef:
    path: str
    alt: str = ""
    caption_text: str = ""
    caption_url: str = ""


@dataclass
class ContentItem:
    kind: str  # "paragraph" | "bullet"
    text: str


@dataclass
class Slide:
    title: str = ""
    subtitle: str = ""
    content: list[ContentItem] = field(default_factory=list)
    code_blocks: list[CodeBlock] = field(default_factory=list)
    images: list[ImageRef] = field(default_factory=list)
    image_before_content: bool = False  # True if first image appeared before any content
    speaker_notes: list[str] = field(default_factory=list)
    is_part_header: bool = False  # # Part headings

    @property
    def bullets(self) -> list[str]:
        return [c.text for c in self.content if c.kind == "bullet"]

    @property
    def paragraphs(self) -> list[str]:
        return [c.text for c in self.content if c.kind == "paragraph"]

    @property
    def has_content(self) -> bool:
        return bool(self.content)


# ---------------------------------------------------------------------------
# Parsing
# ---------------------------------------------------------------------------

_FRONTMATTER_RE = re.compile(r"^---\s*\n(.*?)\n---\s*\n", re.DOTALL)
_COMMENT_RE = re.compile(r"<!--(.*?)-->", re.DOTALL)


def _extract_comments(text: str) -> tuple[str, list[str]]:
    """Extract HTML comments as speaker notes, return cleaned text and notes."""
    notes = []
    for m in _COMMENT_RE.finditer(text):
        note = m.group(1).strip()
        if note:
            notes.append(note)
    cleaned = _COMMENT_RE.sub("", text)
    return cleaned, notes


def _parse_slide_block(block: str) -> Slide:
    """Parse a single slide block into a Slide object."""
    cleaned, notes = _extract_comments(block)
    lines = cleaned.strip().splitlines()

    slide = Slide(speaker_notes=notes)

    in_code = False
    code_lang = ""
    code_lines: list[str] = []
    pending_caption_for: ImageRef | None = None

    for line in lines:
        stripped = line.strip()

        # Blank line: any pending image-caption association ends.
        if not stripped and not in_code:
            pending_caption_for = None
            continue

        # Strip GitHub-style blockquote prefixes (used for [!TIP] callouts).
        if stripped.startswith(">"):
            stripped = stripped[1:].lstrip()
            # Drop the standalone "[!TIP]" / "[!NOTE]" marker lines outright.
            if re.fullmatch(r"\[![A-Z]+\]", stripped):
                continue
            if not stripped:
                continue

        # Code fence open/close
        if stripped.startswith("```"):
            if in_code:
                slide.code_blocks.append(
                    CodeBlock(language=code_lang, code="\n".join(code_lines).strip())
                )
                code_lines = []
                in_code = False
            else:
                in_code = True
                code_lang = stripped[3:].strip()
            continue

        if in_code:
            code_lines.append(line)
            continue

        # Headings
        if stripped.startswith("# "):
            title_text = stripped[2:].strip()
            if title_text.startswith("Part "):
                slide.is_part_header = True
            slide.title = title_text
            continue
        if stripped.startswith("## "):
            # Use ## as the slide title (more common in this deck)
            if not slide.title:
                slide.title = stripped[3:].strip()
            else:
                slide.subtitle = stripped[3:].strip()
            continue

        # Bullets
        if stripped.startswith("- "):
            slide.content.append(ContentItem(kind="bullet", text=stripped[2:]))
            continue

        # Inline images: ![alt](path)
        img_match = re.match(r"!\[(.*?)\]\((.+?)\)\s*$", stripped)
        if img_match:
            ref = ImageRef(alt=img_match.group(1), path=img_match.group(2))
            if not slide.images and not slide.content:
                slide.image_before_content = True
            slide.images.append(ref)
            pending_caption_for = ref
            continue

        # Caption link immediately following an image: [text](url)
        if pending_caption_for is not None and stripped:
            link_match = re.match(r"\[(.+?)\]\((.+?)\)\s*$", stripped)
            if link_match:
                pending_caption_for.caption_text = link_match.group(1)
                pending_caption_for.caption_url = link_match.group(2)
                pending_caption_for = None
                continue
            pending_caption_for = None

        # Standalone bold/text lines (like "** Consider implementing...")
        if stripped.startswith("**") or (stripped and not stripped.startswith("#")):
            # Treat non-empty, non-heading lines as paragraphs
            if stripped:
                slide.content.append(ContentItem(kind="paragraph", text=stripped))
            continue

    return slide


def parse_slides_md(path: Path) -> tuple[PresentationMeta, list[Slide]]:
    """Parse a SLIDES.md file into metadata and slides."""
    text = path.read_text()

    # Extract frontmatter
    raw_meta: dict = {}
    body = text
    m = _FRONTMATTER_RE.match(text)
    if m:
        raw_meta = yaml.safe_load(m.group(1)) or {}
        body = text[m.end() :]

    theme_data = raw_meta.pop("theme", {}) or {}
    theme = ThemeConfig(
        **{k: v for k, v in theme_data.items() if k in ThemeConfig.__dataclass_fields__}
    )
    meta = PresentationMeta(
        title=raw_meta.get("title", "Untitled"),
        subtitle=raw_meta.get("subtitle", ""),
        author=raw_meta.get("author", ""),
        theme=theme,
    )

    # Split on --- slide separators.
    # Must not be inside a code fence. We handle this by tracking fence state.
    slide_texts: list[str] = []
    current_lines: list[str] = []
    in_fence = False

    for line in body.splitlines(keepends=True):
        stripped = line.strip()
        if stripped.startswith("```"):
            in_fence = not in_fence

        if not in_fence and re.match(r"^---\s*$", stripped):
            slide_texts.append("".join(current_lines))
            current_lines = []
        else:
            current_lines.append(line)

    # Last block
    if current_lines:
        slide_texts.append("".join(current_lines))

    slides = []
    for block in slide_texts:
        block = block.strip()
        if not block:
            continue
        slides.append(_parse_slide_block(block))

    return meta, slides


# ---------------------------------------------------------------------------
# Carbon image generation
# ---------------------------------------------------------------------------

# Carbon language MIME types
_CARBON_LANG_MAP = {
    "go": "text/x-go",
    "bash": "application/x-sh",
    "diff": "text/x-diff",
    "python": "text/x-python",
    "yaml": "text/x-yaml",
    "json": "application/json",
    "": "auto",
}

# Carbon settings matching the user-provided URL
_CARBON_PARAMS = {
    "bg": "rgba(171,184,195,0)",
    "t": "vscode",
    "wt": "none",
    "width": "1100",
    "ds": "false",
    "dsyoff": "20px",
    "dsblur": "68px",
    "wc": "false",
    "wa": "false",
    "pv": "30px",
    "ph": "32px",
    "ln": "false",
    "fl": "1",
    "fm": "Hack",
    "fs": "14px",
    "lh": "133%",
    "si": "false",
    "es": "2x",
    "wm": "false",
}


def _carbon_cache_key(code: str, language: str) -> str:
    """Deterministic hash for a code block to use as a cache key."""
    h = hashlib.sha256(f"{language}\n{code}".encode()).hexdigest()[:16]
    return h


def _build_carbon_url(code: str, language: str) -> str:
    """Build a carbon.now.sh URL with the given code and language.

    Always uses "auto" language detection to avoid trailing blank lines that
    some CodeMirror modes (like text/x-diff) add. Diff coloring is applied
    separately via CodeMirror.markText() after the page loads.
    """
    params = dict(_CARBON_PARAMS)
    params["l"] = "auto"
    # Carbon expects the code param to be double-encoded: the browser decodes one
    # layer, then carbon's JavaScript decodes the second.
    params["code"] = quote(code, safe="")
    return f"https://carbon.now.sh/?{urlencode(params, quote_via=quote)}"


def _diff_mark_js(theme: ThemeConfig) -> str:
    """JavaScript to color diff lines via CodeMirror's markText API."""
    return f"""
() => {{
    const cm = document.querySelector('.CodeMirror').CodeMirror;
    const lineCount = cm.lineCount();
    for (let i = 0; i < lineCount; i++) {{
        const text = cm.getLine(i);
        let color = null;
        if (text.startsWith('+')) color = '{theme.diff_add_color}';
        else if (text.startsWith('-')) color = '{theme.diff_remove_color}';
        if (color) {{
            cm.markText(
                {{line: i, ch: 0}},
                {{line: i, ch: text.length}},
                {{css: 'color: ' + color + ' !important'}}
            );
        }}
    }}
}}
"""


async def capture_carbon_images(
    code_blocks: list[CodeBlock],
    cache_dir: Path,
    theme: ThemeConfig,
) -> dict[str, Path]:
    """Download carbon.now.sh PNG exports for all code blocks.

    Uses carbon's built-in export button to download properly rendered PNGs
    at 4x resolution. Images are cached on disk by content hash so unchanged
    code blocks aren't re-downloaded.
    """
    from playwright.async_api import async_playwright

    cache_dir.mkdir(parents=True, exist_ok=True)

    # Determine which blocks need downloading vs are already cached
    results: dict[str, Path] = {}
    to_capture: list[tuple[CodeBlock, str, Path]] = []

    for cb in code_blocks:
        key = _carbon_cache_key(cb.code, cb.language)
        img_path = cache_dir / f"{key}.png"
        if img_path.exists():
            results[key] = img_path
        else:
            to_capture.append((cb, key, img_path))

    if not to_capture:
        return results

    print(f"  Downloading {len(to_capture)} code images from carbon.now.sh ({len(results)} cached)...")

    async with async_playwright() as pw:
        browser = await pw.chromium.launch()
        page = await browser.new_page(viewport={"width": 1400, "height": 1000})

        for i, (cb, key, img_path) in enumerate(to_capture):
            url = _build_carbon_url(cb.code, cb.language)
            await page.goto(url, wait_until="networkidle")
            # Wait for the code container to render
            await page.wait_for_selector(".export-container", state="visible", timeout=15000)
            await page.wait_for_timeout(1000)

            # Apply diff line coloring via CodeMirror markText
            if cb.language == "diff":
                await page.evaluate(_diff_mark_js(theme))
                await page.wait_for_timeout(300)

            # Open the export dropdown
            await page.click("#export-menu")
            await page.wait_for_timeout(500)

            # Select 4x export size
            await page.click("button:has-text('4x')")
            await page.wait_for_timeout(200)

            # Download the PNG via carbon's export button
            async with page.expect_download(timeout=15000) as download_info:
                await page.click("#export-png")
            download = await download_info.value
            await download.save_as(str(img_path))

            print(f"    [{i + 1}/{len(to_capture)}] {cb.language or 'text'}: {key}.png")

        await browser.close()

    results.update({key: img_path for _, key, img_path in to_capture})
    return results


def collect_all_code_blocks(slides: list[Slide]) -> list[CodeBlock]:
    """Collect all unique code blocks across all slides."""
    seen: set[str] = set()
    blocks: list[CodeBlock] = []
    for s in slides:
        for cb in s.code_blocks:
            key = _carbon_cache_key(cb.code, cb.language)
            if key not in seen:
                seen.add(key)
                blocks.append(cb)
    return blocks


# ---------------------------------------------------------------------------
# PPTX rendering helpers
# ---------------------------------------------------------------------------


def _hex_to_rgb(hex_color: str) -> RGBColor:
    h = hex_color.lstrip("#")[:6]
    return RGBColor(int(h[0:2], 16), int(h[2:4], 16), int(h[4:6], 16))


def _set_slide_bg(slide, hex_color: str) -> None:
    fill = slide.background.fill
    fill.solid()
    fill.fore_color.rgb = _hex_to_rgb(hex_color)


def _add_textbox(
    slide,
    left: float,
    top: float,
    width: float,
    height: float,
):
    return slide.shapes.add_textbox(
        Inches(left), Inches(top), Inches(width), Inches(height)
    )


_INLINE_MD_RE = re.compile(
    r"(\*\*.*?\*\*|`[^`]+`|~~.*?~~|\[[^\]]+?\]\([^)]+?\)|\*[^*\s][^*]*?\*)"
)


def _add_formatted_runs(
    paragraph, text: str, *,
    font_name: str, font_size: int, color: str, bold: bool = False,
    code_font: str = "Courier New", link_color: str | None = None,
) -> None:
    """Add runs to a paragraph, parsing inline markdown:
    **bold**, *italic*, `code`, ~~strike~~, [text](url).
    """
    parts = _INLINE_MD_RE.split(text)
    for part in parts:
        if not part:
            continue
        run = paragraph.add_run()

        # Order matters: handle longer/more-specific markers first.
        if part.startswith("**") and part.endswith("**") and len(part) >= 4:
            run.text = part[2:-2]
            run.font.bold = True
        elif part.startswith("~~") and part.endswith("~~") and len(part) >= 4:
            run.text = part[2:-2]
            # python-pptx doesn't expose strikethrough cleanly; emulate with italic+grey.
            run.font.italic = True
        elif part.startswith("`") and part.endswith("`") and len(part) >= 2:
            run.text = part[1:-1]
            run.font.name = code_font
            run.font.size = Pt(max(font_size - 2, 10))
            run.font.color.rgb = _hex_to_rgb(color)
            continue
        elif part.startswith("[") and "](" in part and part.endswith(")"):
            close = part.index("](")
            run.text = part[1:close]
            url = part[close + 2:-1]
            run.font.color.rgb = _hex_to_rgb(link_color or color)
            try:
                run.hyperlink.address = url
            except Exception:
                pass
            run.font.name = font_name
            run.font.size = Pt(font_size)
            continue
        elif part.startswith("*") and part.endswith("*") and len(part) >= 2:
            run.text = part[1:-1]
            run.font.italic = True
        else:
            run.text = part
            run.font.bold = bold

        run.font.name = font_name
        run.font.size = Pt(font_size)
        run.font.color.rgb = _hex_to_rgb(color)


def _set_speaker_notes(slide, notes: list[str]) -> None:
    """Set speaker notes on a slide."""
    if not notes:
        return
    notes_slide = slide.notes_slide
    tf = notes_slide.notes_text_frame
    tf.text = "\n\n".join(notes)


# ---------------------------------------------------------------------------
# Slide renderers
# ---------------------------------------------------------------------------


def _render_title_slide(prs: Presentation, slide_data: Slide, meta: PresentationMeta, **_) -> None:
    theme = meta.theme
    slide = prs.slides.add_slide(prs.slide_layouts[6])  # blank
    _set_slide_bg(slide, theme.background_color)

    # Reserve right column for an image (e.g. follow-along QR).
    has_image = bool(slide_data.images and Path(slide_data.images[0].path).exists())
    text_left = 0.6
    text_width = 8.0 if has_image else 11.33

    # Title
    txBox = _add_textbox(slide, text_left, 0.6, text_width, 1.2)
    tf = txBox.text_frame
    tf.word_wrap = True
    p = tf.paragraphs[0]
    p.alignment = PP_ALIGN.LEFT
    _add_formatted_runs(
        p, slide_data.title or meta.title,
        font_name=theme.font_heading, font_size=44, color=theme.title_color, bold=True,
    )

    # Subtitle
    subtitle = slide_data.subtitle or meta.subtitle
    if subtitle:
        txBox = _add_textbox(slide, text_left, 1.9, text_width, 1.0)
        tf = txBox.text_frame
        tf.word_wrap = True
        p = tf.paragraphs[0]
        p.alignment = PP_ALIGN.LEFT
        _add_formatted_runs(p, subtitle, font_name=theme.font_body, font_size=24, color=theme.text_color)

    # Content (paragraphs + bullets in source order)
    if slide_data.content:
        top = 3.0 if subtitle else 1.9
        _render_content_box(slide, slide_data.content, meta, text_left + 0.3, top, text_width - 0.3, 3.5)

    # Right-side image
    if has_image:
        img = slide_data.images[0]
        img_left, img_top, img_h = 9.2, 2.0, 3.5
        slide.shapes.add_picture(str(Path(img.path)), Inches(img_left), Inches(img_top), height=Inches(img_h))
        if img.caption_text:
            cap_top = img_top + img_h + 0.1
            txBox = _add_textbox(slide, img_left - 0.5, cap_top, 4.5, 0.5)
            ctf = txBox.text_frame
            ctf.word_wrap = True
            cp = ctf.paragraphs[0]
            cp.alignment = PP_ALIGN.CENTER
            # Split "Follow along: jacob.work/100TM" into prefix + linked URL.
            # Anything matching the caption_url's host/path becomes the hyperlink.
            link_start = img.caption_text.find(img.caption_url.split("//")[-1].rstrip("/"))
            if img.caption_url and link_start == -1:
                # Fallback: link the whole caption.
                link_start = 0
            prefix = img.caption_text[:link_start] if link_start > 0 else ""
            link_part = img.caption_text[link_start:] if link_start >= 0 else img.caption_text
            if prefix:
                _add_formatted_runs(cp, prefix, font_name=theme.font_body, font_size=14, color=theme.text_color)
            link_run = cp.add_run()
            link_run.text = link_part
            link_run.font.name = theme.font_body
            link_run.font.size = Pt(14)
            link_run.font.color.rgb = _hex_to_rgb(theme.accent_color)
            link_run.hyperlink.address = img.caption_url

    # Author
    if meta.author:
        txBox = _add_textbox(slide, 1.0, 6.7, 11.33, 0.5)
        tf = txBox.text_frame
        p = tf.paragraphs[0]
        p.alignment = PP_ALIGN.LEFT
        run = p.add_run()
        run.text = meta.author
        run.font.name = theme.font_body
        run.font.size = Pt(16)
        run.font.color.rgb = _hex_to_rgb(theme.text_color)

    _set_speaker_notes(slide, slide_data.speaker_notes)


def _render_section_slide(prs: Presentation, slide_data: Slide, meta: PresentationMeta, **_) -> None:
    theme = meta.theme
    slide = prs.slides.add_slide(prs.slide_layouts[6])
    _set_slide_bg(slide, theme.accent_color)

    # Title (left-aligned)
    txBox = _add_textbox(slide, 0.6, 1.4, 12.13, 1.2)
    tf = txBox.text_frame
    tf.word_wrap = True
    p = tf.paragraphs[0]
    p.alignment = PP_ALIGN.LEFT
    _add_formatted_runs(p, slide_data.title, font_name=theme.font_heading, font_size=40, color="#FFFFFF", bold=True)

    paragraphs = slide_data.paragraphs
    bullets = slide_data.bullets

    # Lead paragraphs (no bullet dots, larger body text), left-aligned under title.
    next_top = 2.8
    if paragraphs:
        para_items = [ContentItem(kind="paragraph", text=t) for t in paragraphs]
        para_height = min(len(paragraphs) * 0.6 + 0.2, 2.2)
        _render_content_box(
            slide, para_items, meta, 0.6, next_top, 12.13, para_height,
            color_override="#FFFFFF", align=PP_ALIGN.LEFT, font_size=22,
        )
        next_top += para_height + 0.2

    # Bullets in a tight left-aligned group under the lead.
    if bullets:
        bullet_items = [ContentItem(kind="bullet", text=t) for t in bullets]
        bullet_height = min(len(bullets) * 0.45 + 0.2, 7.4 - next_top)
        _render_content_box(
            slide, bullet_items, meta, 1.0, next_top, 11.73, bullet_height,
            color_override="#FFFFFF", align=PP_ALIGN.LEFT, font_size=20,
        )

    _set_speaker_notes(slide, slide_data.speaker_notes)


def _render_content_slide(
    prs: Presentation,
    slide_data: Slide,
    meta: PresentationMeta,
    carbon_images: dict[str, Path],
) -> None:
    theme = meta.theme
    slide = prs.slides.add_slide(prs.slide_layouts[6])
    _set_slide_bg(slide, theme.background_color)

    # Title
    if slide_data.title:
        txBox = _add_textbox(slide, 0.6, 0.3, 12.0, 1.0)
        tf = txBox.text_frame
        tf.word_wrap = True
        p = tf.paragraphs[0]
        _add_formatted_runs(
            p, slide_data.title,
            font_name=theme.font_heading, font_size=28, color=theme.title_color, bold=True,
        )

    content_top = 1.5
    content_bottom = content_top  # updated by branches below
    items = slide_data.content
    has_items = bool(items)
    has_code = bool(slide_data.code_blocks)
    has_image = bool(slide_data.images and Path(slide_data.images[0].path).exists())

    # If the markdown placed the image before any content, render it first.
    if has_image and slide_data.image_before_content and not has_code:
        img_path = Path(slide_data.images[0].path)
        try:
            from PIL import Image as _PILImage
            with _PILImage.open(img_path) as im:
                aspect = im.width / im.height if im.height else 1.0
        except Exception:
            aspect = 1.0
        # Reserve up to 4.5" of height for the image; leave room for text below.
        reserved_for_text = 1.5 if has_items else 0.0
        max_img_h = 7.5 - content_top - reserved_for_text - 0.2
        img_h = min(4.5, max_img_h)
        img_w = min(12.0, img_h * aspect)
        img_left = 0.6 + (12.0 - img_w) / 2
        slide.shapes.add_picture(
            str(img_path), Inches(img_left), Inches(content_top), width=Inches(img_w),
        )
        content_top += img_h + 0.2
        content_bottom = content_top
        has_image = False  # consumed

    if has_items and has_code:
        # Content on top, code below.  Estimate prose space using the same
        # char-wrap heuristic as the items-only branch.
        text_font = 18
        cpl = max(20, int(70 * 22 / text_font))
        text_lines = sum(max(1, -(-len(c.text) // cpl)) for c in items)
        items_height = min(7.5 - content_top - 1.5, max(0.6, text_lines * 0.32 + 0.2))
        _render_content_box(slide, items, meta, 0.6, content_top, 12.0, items_height, font_size=text_font)
        code_top = content_top + items_height
        remaining = 7.5 - code_top
        _render_code_region(slide, slide_data.code_blocks, meta, 0.6, code_top, 12.0, remaining, carbon_images)
        content_bottom = code_top + remaining
    elif has_items:
        # Estimate visual lines per item. Calibrated: 12" wide @ 22pt fits ~70 chars
        # in the body font, so each pt scales the chars/line ratio inversely.
        def chars_per_line(font_pt: int) -> int:
            return max(20, int(70 * 22 / font_pt))

        def visual_lines(it: ContentItem, cpl: int) -> int:
            return max(1, -(-len(it.text) // cpl))

        # Try font sizes from largest to smallest; pick the largest that fits.
        candidates = [28, 22, 20, 18]
        has_paragraph = any(c.kind == "paragraph" for c in items)
        # Cap paragraphs at 22pt to keep prose readable.
        if has_paragraph:
            candidates = [22, 20, 18]

        font_size = candidates[-1]
        for fs in candidates:
            cpl = chars_per_line(fs)
            tl = sum(visual_lines(c, cpl) for c in items)
            line_h = fs * 1.4 / 72
            needed = tl * line_h + 0.3
            available = 7.5 - content_top - 0.3
            if needed <= available:
                font_size = fs
                total_lines = tl
                est_height = needed
                break
        else:
            cpl = chars_per_line(font_size)
            total_lines = sum(visual_lines(c, cpl) for c in items)
            est_height = total_lines * font_size * 1.4 / 72 + 0.3

        available = 7.5 - content_top - 0.3
        # Re-center only if content uses ≤50% of available area.
        if est_height < available * 0.5:
            content_top = content_top + (available - est_height) / 2
        box_h = min(max(est_height, 1.5), 7.5 - content_top - 0.2)
        _render_content_box(
            slide, items, meta, 0.6, content_top, 12.0, box_h,
            font_size=font_size,
        )
        content_bottom = content_top + box_h
    elif has_code:
        code_h = 7.5 - content_top
        _render_code_region(slide, slide_data.code_blocks, meta, 0.6, content_top, 12.0, code_h, carbon_images)
        content_bottom = content_top + code_h

    if has_image:
        img_path = Path(slide_data.images[0].path)
        # Position the image after any text content so they don't overlap.
        img_top = content_bottom + (0.2 if has_items or has_code else 0.0)
        # Pick a width that fits within the remaining vertical room.
        try:
            from PIL import Image
            with Image.open(img_path) as im:
                aspect = im.width / im.height if im.height else 1.0
        except Exception:
            aspect = 1.0
        avail_h = max(0.5, 7.5 - img_top - 0.2)
        img_w = min(12.0, avail_h * aspect)
        img_left = 0.6 + (12.0 - img_w) / 2
        slide.shapes.add_picture(
            str(img_path), Inches(img_left), Inches(img_top), width=Inches(img_w),
        )

    _set_speaker_notes(slide, slide_data.speaker_notes)


def _render_content_box(
    slide,
    items: list[ContentItem],
    meta: PresentationMeta,
    left: float,
    top: float,
    width: float,
    height: float,
    *,
    color_override: str | None = None,
    align: int | None = None,
    font_size: int = 18,
) -> None:
    theme = meta.theme
    txBox = _add_textbox(slide, left, top, width, height)
    tf = txBox.text_frame
    tf.word_wrap = True

    text_color = color_override or theme.text_color
    accent = color_override or theme.accent_color
    link_color = theme.accent_color

    for i, item in enumerate(items):
        p = tf.paragraphs[0] if i == 0 else tf.add_paragraph()
        p.space_after = Pt(6)
        p.space_before = Pt(2)
        if align is not None:
            p.alignment = align

        if item.kind == "bullet":
            brun = p.add_run()
            brun.text = "\u2022  "
            brun.font.name = theme.font_body
            brun.font.size = Pt(font_size)
            brun.font.color.rgb = _hex_to_rgb(accent)

        _add_formatted_runs(
            p, item.text,
            font_name=theme.font_body, font_size=font_size, color=text_color,
            code_font=theme.code_font, link_color=link_color,
        )


def _render_bullets_box(
    slide,
    bullets: list[str],
    meta: PresentationMeta,
    left: float,
    top: float,
    width: float,
    height: float,
    *,
    color_override: str | None = None,
) -> None:
    items = [ContentItem(kind="bullet", text=b) for b in bullets]
    _render_content_box(slide, items, meta, left, top, width, height, color_override=color_override)


# ---------------------------------------------------------------------------
# Code rendering (carbon images)
# ---------------------------------------------------------------------------


def _render_code_region(
    slide,
    code_blocks: list[CodeBlock],
    meta: PresentationMeta,
    left: float,
    top: float,
    width: float,
    max_height: float,
    carbon_images: dict[str, Path],
) -> None:
    """Render code blocks as carbon screenshot images."""
    _render_code_images(slide, code_blocks, left, top, width, max_height, carbon_images)


def _render_code_images(
    slide,
    code_blocks: list[CodeBlock],
    left: float,
    top: float,
    width: float,
    max_height: float,
    carbon_images: dict[str, Path],
) -> None:
    """Embed carbon screenshot images for code blocks."""
    from PIL import Image

    current_top = top
    gap = 0.0

    # Measure all images to calculate proportional sizing
    measurements: list[tuple[CodeBlock, Path, int, int]] = []
    for cb in code_blocks:
        key = _carbon_cache_key(cb.code, cb.language)
        img_path = carbon_images.get(key)
        if not img_path:
            continue
        with Image.open(img_path) as img:
            measurements.append((cb, img_path, img.width, img.height))

    if not measurements:
        return

    total_aspect_height = sum(h / w for _, _, w, h in measurements)
    usable_height = max_height - gap * (len(measurements) - 1)

    for cb, img_path, img_w, img_h in measurements:
        # Proportional height allocation based on image aspect ratios
        aspect = img_h / img_w
        block_height = (aspect / total_aspect_height) * usable_height

        # Fit to width, then check if height overflows
        display_width = width
        display_height = display_width * (img_h / img_w)

        if display_height > block_height:
            display_height = block_height
            display_width = display_height * (img_w / img_h)

        # Center horizontally
        x_offset = left + (width - display_width) / 2

        slide.shapes.add_picture(
            str(img_path),
            Inches(x_offset),
            Inches(current_top),
            Inches(display_width),
            Inches(display_height),
        )

        current_top += display_height + gap


# ---------------------------------------------------------------------------
# Main generation
# ---------------------------------------------------------------------------


def generate_pptx(
    meta: PresentationMeta,
    slides: list[Slide],
    output: Path,
    carbon_images: dict[str, Path],
) -> None:
    prs = Presentation()
    prs.slide_width = Inches(13.333)
    prs.slide_height = Inches(7.5)

    for i, s in enumerate(slides):
        if i == 0:
            _render_title_slide(prs, s, meta)
        elif s.is_part_header:
            _render_section_slide(prs, s, meta)
        elif not s.content and not s.code_blocks and not s.images and s.title:
            if not s.subtitle and len(s.title) < 60:
                _render_section_slide(prs, s, meta)
            else:
                _render_content_slide(prs, s, meta, carbon_images)
        else:
            _render_content_slide(prs, s, meta, carbon_images)

    prs.save(str(output))


def main() -> None:
    parser = argparse.ArgumentParser(description="Generate PPTX from SLIDES.md")
    parser.add_argument("input", nargs="?", default="SLIDES.md", help="Input markdown file")
    parser.add_argument("-o", "--output", default=None, help="Output .pptx path")
    args = parser.parse_args()

    input_path = Path(args.input)
    if not input_path.exists():
        print(f"Error: {input_path} not found", file=sys.stderr)
        sys.exit(1)

    output_path = Path(args.output) if args.output else input_path.with_suffix(".pptx")

    meta, slides = parse_slides_md(input_path)
    print(f"Parsed {len(slides)} slides from {input_path}")

    cache_dir = input_path.parent / ".slide_images"
    all_code = collect_all_code_blocks(slides)
    carbon_images: dict[str, Path] = {}
    if all_code:
        carbon_images = asyncio.run(capture_carbon_images(all_code, cache_dir, meta.theme))
        print(f"  {len(carbon_images)} code block images ready")

    generate_pptx(meta, slides, output_path, carbon_images)
    print(f"Generated {output_path}")


if __name__ == "__main__":
    main()
