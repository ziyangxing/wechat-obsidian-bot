"""WeChat article extractor using feedgrab.
Usage: python extract_article.py <url>
Outputs JSON with title, content, author, url to stdout.
"""
import sys
import json
import asyncio

# Fix Windows GBK encoding issue (ignore if not supported)
try:
    sys.stdout.reconfigure(encoding='utf-8', errors='replace')
except Exception:
    pass

from feedgrab.reader import UniversalReader


async def extract(url: str) -> dict:
    reader = UniversalReader()
    try:
        result = await reader.read(url)
        return {
            "title": result.title or "",
            "content": result.content or "",
            "author": result.source_name or "",
            "url": url,
        }
    except Exception as e:
        return {"title": "", "content": "", "author": "", "url": url, "error": str(e)}


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print(json.dumps({"error": "No URL provided"}))
        sys.exit(1)

    result = asyncio.run(extract(sys.argv[1]))
    print(json.dumps(result, ensure_ascii=False))
