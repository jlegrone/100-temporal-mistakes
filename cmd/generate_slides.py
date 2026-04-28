#!/usr/bin/env python3
"""Generate a PowerPoint presentation from SLIDES.md.

SLIDES.md format:
  - Optional YAML frontmatter (between --- delimiters) for presentation metadata
  - Slides separated by --- on its own line (not indented, not inside code fences)
  - Markdown content: # Title, ## Subtitle, - bullets, **bold**, *italic*
  - Code blocks with language hints: ```go, ```bash, ```diff
  - HTML comments (<!-- ... -->) become speaker notes

Usage:
    uv run --with python-pptx --with pyyaml --with playwright \
        generate_slides.py [SLIDES.md] [-o output.pptx]

    # Skip carbon image generation (use text-based code blocks):
    uv run --with python-pptx --with pyyaml \
        generate_slides.py --no-carbon [SLIDES.md] [-o output.pptx]

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
from pptx.enum.shapes import MSO_SHAPE
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
class Slide:
    title: str = ""
    subtitle: str = ""
    bullets: list[str] = field(default_factory=list)
    code_blocks: list[CodeBlock] = field(default_factory=list)
    speaker_notes: list[str] = field(default_factory=list)
    is_part_header: bool = False  # # Part headings


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

    for line in lines:
        stripped = line.strip()

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
            slide.bullets.append(stripped[2:])
            continue

        # Standalone bold/text lines (like "** Consider implementing...")
        if stripped.startswith("**") or (stripped and not stripped.startswith("#")):
            # Treat non-empty, non-heading lines as bullets
            if stripped:
                slide.bullets.append(stripped)
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


def _add_formatted_runs(
    paragraph, text: str, *, font_name: str, font_size: int, color: str, bold: bool = False
) -> None:
    """Add runs to a paragraph, parsing inline **bold** and *italic* markers."""
    parts = re.split(r"(\*\*.*?\*\*|\*[^*]+?\*)", text)
    for part in parts:
        if not part:
            continue
        run = paragraph.add_run()
        if part.startswith("**") and part.endswith("**"):
            run.text = part[2:-2]
            run.font.bold = True
        elif part.startswith("*") and part.endswith("*"):
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

    # Title
    txBox = _add_textbox(slide, 1.0, 2.2, 11.33, 1.5)
    tf = txBox.text_frame
    tf.word_wrap = True
    p = tf.paragraphs[0]
    p.alignment = PP_ALIGN.CENTER
    _add_formatted_runs(
        p, slide_data.title or meta.title,
        font_name=theme.font_heading, font_size=44, color=theme.title_color, bold=True,
    )

    # Subtitle
    subtitle = slide_data.subtitle or meta.subtitle
    if subtitle:
        txBox = _add_textbox(slide, 1.0, 3.8, 11.33, 1.0)
        tf = txBox.text_frame
        tf.word_wrap = True
        p = tf.paragraphs[0]
        p.alignment = PP_ALIGN.CENTER
        _add_formatted_runs(p, subtitle, font_name=theme.font_body, font_size=24, color=theme.text_color)

    # Bullets (if the title slide has them)
    if slide_data.bullets:
        top = 4.8 if subtitle else 3.8
        _render_bullets_box(slide, slide_data.bullets, meta, 1.5, top, 10.33, 3.0)

    # Author
    if meta.author:
        txBox = _add_textbox(slide, 1.0, 6.0, 11.33, 0.5)
        tf = txBox.text_frame
        p = tf.paragraphs[0]
        p.alignment = PP_ALIGN.CENTER
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

    y = 2.2
    txBox = _add_textbox(slide, 1.0, y, 11.33, 1.8)
    tf = txBox.text_frame
    tf.word_wrap = True
    p = tf.paragraphs[0]
    p.alignment = PP_ALIGN.CENTER
    _add_formatted_runs(p, slide_data.title, font_name=theme.font_heading, font_size=40, color="#FFFFFF", bold=True)

    if slide_data.bullets:
        _render_bullets_box(slide, slide_data.bullets, meta, 1.5, 4.2, 10.33, 3.0, color_override="#FFFFFF")

    _set_speaker_notes(slide, slide_data.speaker_notes)


def _render_content_slide(
    prs: Presentation,
    slide_data: Slide,
    meta: PresentationMeta,
    carbon_images: dict[str, Path] | None = None,
) -> None:
    theme = meta.theme
    slide = prs.slides.add_slide(prs.slide_layouts[6])
    _set_slide_bg(slide, theme.background_color)

    # Title
    if slide_data.title:
        txBox = _add_textbox(slide, 0.6, 0.3, 12.0, 0.8)
        tf = txBox.text_frame
        tf.word_wrap = True
        p = tf.paragraphs[0]
        _add_formatted_runs(
            p, slide_data.title,
            font_name=theme.font_heading, font_size=28, color=theme.title_color, bold=True,
        )

    content_top = 1.3
    has_bullets = bool(slide_data.bullets)
    has_code = bool(slide_data.code_blocks)

    if has_bullets and has_code:
        # Bullets on top, code below
        bullet_height = min(len(slide_data.bullets) * 0.4 + 0.2, 2.5)
        _render_bullets_box(slide, slide_data.bullets, meta, 0.6, content_top, 12.0, bullet_height)
        code_top = content_top + bullet_height + 0.15
        remaining = 7.5 - code_top - 0.2
        _render_code_region(slide, slide_data.code_blocks, meta, 0.6, code_top, 12.0, remaining, carbon_images)
    elif has_bullets:
        _render_bullets_box(slide, slide_data.bullets, meta, 0.6, content_top, 12.0, 5.5)
    elif has_code:
        _render_code_region(slide, slide_data.code_blocks, meta, 0.6, content_top, 12.0, 5.8, carbon_images)

    _set_speaker_notes(slide, slide_data.speaker_notes)


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
    theme = meta.theme
    txBox = _add_textbox(slide, left, top, width, height)
    tf = txBox.text_frame
    tf.word_wrap = True

    text_color = color_override or theme.text_color
    accent = color_override or theme.accent_color

    for i, bullet in enumerate(bullets):
        p = tf.paragraphs[0] if i == 0 else tf.add_paragraph()
        p.space_after = Pt(6)
        p.space_before = Pt(2)

        # Bullet character
        brun = p.add_run()
        brun.text = "\u2022  "
        brun.font.name = theme.font_body
        brun.font.size = Pt(18)
        brun.font.color.rgb = _hex_to_rgb(accent)

        _add_formatted_runs(p, bullet, font_name=theme.font_body, font_size=18, color=text_color)


# ---------------------------------------------------------------------------
# Code rendering (carbon images or text fallback)
# ---------------------------------------------------------------------------


def _render_code_region(
    slide,
    code_blocks: list[CodeBlock],
    meta: PresentationMeta,
    left: float,
    top: float,
    width: float,
    max_height: float,
    carbon_images: dict[str, Path] | None,
) -> None:
    """Render code blocks as carbon images if available, otherwise as styled text."""
    if carbon_images:
        _render_code_images(slide, code_blocks, left, top, width, max_height, carbon_images)
    else:
        _render_code_text(slide, code_blocks, meta, left, top, width, max_height)


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
    gap = 0.1

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


def _render_code_text(
    slide,
    code_blocks: list[CodeBlock],
    meta: PresentationMeta,
    left: float,
    top: float,
    width: float,
    max_height: float,
) -> None:
    """Render code blocks as styled text (fallback when carbon images not available)."""
    theme = meta.theme
    current_top = top

    total_lines = sum(len(cb.code.splitlines()) for cb in code_blocks)
    if total_lines == 0:
        return

    for cb in code_blocks:
        lines = cb.code.splitlines()
        proportion = len(lines) / total_lines if total_lines > 0 else 1.0
        block_height = max(0.5, proportion * (max_height - 0.1 * len(code_blocks)))
        block_height = min(block_height, top + max_height - current_top)

        if block_height <= 0:
            break

        # Background rectangle
        bg = slide.shapes.add_shape(
            MSO_SHAPE.ROUNDED_RECTANGLE,
            Inches(left), Inches(current_top), Inches(width), Inches(block_height),
        )
        bg.fill.solid()
        bg.fill.fore_color.rgb = _hex_to_rgb(theme.code_bg_color)
        bg.line.fill.background()
        bg.adjustments[0] = 0.02

        # Code text
        txBox = _add_textbox(slide, left + 0.2, current_top + 0.08, width - 0.4, block_height - 0.16)
        tf = txBox.text_frame
        tf.word_wrap = False

        is_diff = cb.language == "diff"
        font_size = _auto_font_size(lines, width - 0.4)

        for j, code_line in enumerate(lines):
            p = tf.paragraphs[0] if j == 0 else tf.add_paragraph()
            p.alignment = PP_ALIGN.LEFT
            p.space_before = Pt(0)
            p.space_after = Pt(0)
            p.line_spacing = Pt(font_size + 2)

            run = p.add_run()
            run.font.name = theme.code_font
            run.font.size = Pt(font_size)

            if is_diff:
                if code_line.startswith("+"):
                    run.font.color.rgb = _hex_to_rgb(theme.diff_add_color)
                elif code_line.startswith("-"):
                    run.font.color.rgb = _hex_to_rgb(theme.diff_remove_color)
                else:
                    run.font.color.rgb = _hex_to_rgb(theme.code_text_color)
            else:
                run.font.color.rgb = _hex_to_rgb(theme.code_text_color)

            run.text = code_line

        current_top += block_height + 0.1


def _auto_font_size(lines: list[str], width_inches: float) -> int:
    """Pick a font size that fits the code in the available space."""
    if not lines:
        return 11
    max_chars = max((len(line) for line in lines), default=0)
    chars_per_inch_at_11 = 13.0
    available_chars = width_inches * chars_per_inch_at_11
    if max_chars <= available_chars:
        if len(lines) <= 12:
            return 11
        if len(lines) <= 20:
            return 10
        return 9
    ratio = available_chars / max_chars
    return max(7, int(11 * ratio))


# ---------------------------------------------------------------------------
# Main generation
# ---------------------------------------------------------------------------


def generate_pptx(
    meta: PresentationMeta,
    slides: list[Slide],
    output: Path,
    carbon_images: dict[str, Path] | None = None,
) -> None:
    prs = Presentation()
    prs.slide_width = Inches(13.333)
    prs.slide_height = Inches(7.5)

    for i, s in enumerate(slides):
        if i == 0:
            _render_title_slide(prs, s, meta)
        elif s.is_part_header:
            _render_section_slide(prs, s, meta)
        elif not s.bullets and not s.code_blocks and s.title:
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
    parser.add_argument(
        "--no-carbon", action="store_true",
        help="Skip carbon.now.sh image generation; use text-based code blocks instead",
    )
    args = parser.parse_args()

    input_path = Path(args.input)
    if not input_path.exists():
        print(f"Error: {input_path} not found", file=sys.stderr)
        sys.exit(1)

    output_path = Path(args.output) if args.output else input_path.with_suffix(".pptx")

    meta, slides = parse_slides_md(input_path)
    print(f"Parsed {len(slides)} slides from {input_path}")

    carbon_images: dict[str, Path] | None = None
    if not args.no_carbon:
        try:
            import playwright  # noqa: F401
        except ImportError:
            print(
                "Error: playwright is required for carbon screenshots.\n"
                "  Install: uv run --with playwright python -m playwright install chromium\n"
                "  Or use --no-carbon for text-based code blocks.",
                file=sys.stderr,
            )
            sys.exit(1)

        cache_dir = input_path.parent / ".slide_images"
        all_code = collect_all_code_blocks(slides)
        if all_code:
            carbon_images = asyncio.run(capture_carbon_images(all_code, cache_dir, meta.theme))
            print(f"  {len(carbon_images)} code block images ready")

    generate_pptx(meta, slides, output_path, carbon_images)
    print(f"Generated {output_path}")


if __name__ == "__main__":
    main()
