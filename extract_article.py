"""WeChat article extractor using feedgrab.
Usage: python extract_article.py <url>
Outputs JSON with title, content, author, url, cover_image, all_images to stdout.

Note on images: feedgrab's HTML->Markdown converter strips <img> tags from
table cells. Images embedded in prose are captured; table images are not.
The article URL is always saved for full access.
"""
import sys
import json
import asyncio
import re

try:
    sys.stdout.reconfigure(encoding='utf-8', errors='replace')
except Exception:
    pass

from feedgrab.reader import UniversalReader
from feedgrab.fetchers.wechat import fetch_wechat


async def extract(url: str) -> dict:
    reader = UniversalReader()
    result = await reader.read(url)

    # Get extra metadata from the raw fetcher (cover image, etc.)
    raw = await fetch_wechat(url)

    all_images = []
    cover = raw.get("cover_image", "")
    if cover and cover.startswith("http"):
        all_images.append(cover)

    # Also scan markdown for any ![](url) that feedgrab preserved
    for m in re.finditer(r'!\[[^\]]*\]\(([^)]+)\)', result.content):
        url_match = m.group(1)
        if url_match.startswith("http") and url_match not in all_images:
            all_images.append(url_match)

    # And scan for naked image URLs
    for m in re.finditer(r'(https?://[^\s<>"]+\.(?:png|jpg|jpeg|gif|webp)(?:\?[^\s<>"]*)?)', result.content, re.I):
        url_match = m.group(1)
        if url_match not in all_images:
            all_images.append(url_match)

    return {
        "title": result.title or "",
        "content": result.content or "",
        "author": result.source_name or "",
        "url": url,
        "all_images": all_images,
    }


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print(json.dumps({"error": "No URL provided"}))
        sys.exit(1)

    result = asyncio.run(extract(sys.argv[1]))
    print(json.dumps(result, ensure_ascii=False))
