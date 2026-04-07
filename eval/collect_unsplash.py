#!/usr/bin/env python3
import os
import time
import requests
from pathlib import Path
from urllib.parse import urlencode

CATEGORIES: list[str] = [
    "graduation", "wedding", "birthday", "party", "kids", "family", "nails", "cake",
    "child", "thailand", "hiking", "sea", "skiing", "night city driving", "document",
    "dog", "cat", "elephant", "snake", "friends", "school", "park", "sea", "picnic",
    "sunset", "selfie", "group photo", "group photo at a dining table", "a boy", "a girl",
]

# Remove duplicates while preserving order (note: "sea" appears twice in the list)
_seen = set()
CATEGORIES = [c for c in CATEGORIES if not (c in _seen or _seen.add(c))]

IMAGES_PER_CATEGORY = int(os.environ.get("UNSPLASH_IMAGES_PER_CATEGORY", "100"))
CREATED_AFTER_ISO = os.environ.get("UNSPLASH_CREATED_AFTER", "2025-10-16T00:00:00Z")
OUT_DIR = Path(os.environ.get("UNSPLASH_OUT_DIR", "./eval_data"))
UNSPLASH_ACCESS_KEY = os.environ.get("UNSPLASH_ACCESS_KEY", "ifFnaQ6EO3ZTPebf6ezXd-j6gfULNpfrk8mWaZt16IE")
UNSPLASH_APP_ID = os.environ.get("UNSPLASH_APP_ID", "877151")

BASE_URL = "https://api.unsplash.com/search/photos"
PER_PAGE = 30  # Unsplash maximum per_page

HEADERS = {
    "Accept-Version": "v1",
    "Authorization": f"Client-ID {UNSPLASH_ACCESS_KEY}",
}


def ratelimit_sleep(resp: requests.Response):
    try:
        remaining = int(resp.headers.get("X-Ratelimit-Remaining", "1"))
        reset = int(resp.headers.get("X-Ratelimit-Reset", str(int(time.time()) + 60)))
    except ValueError:
        remaining, reset = 1, int(time.time()) + 60

    if remaining <= 1:
        now = int(time.time())
        sleep_for = max(0, reset - now) + 1
        print(f"Rate limit reached. Sleeping for {sleep_for}s until reset...")
        time.sleep(sleep_for)


def search_page(query: str, page: int) -> dict:
    params = {
        "query": query,
        "page": page,
        "per_page": PER_PAGE,
        "order_by": "latest",
        # Unlike official filter, Unsplash API does not support created_at filter directly.
        # We'll filter the client-side based on the 'created_at' field.
    }
    url = f"{BASE_URL}?{urlencode(params)}"
    last_err = None
    for attempt in range(5):
        try:
            resp = requests.get(url, headers=HEADERS, timeout=30)
            if resp.status_code == 429:
                ratelimit_sleep(resp)
                continue
            resp.raise_for_status()
            ratelimit_sleep(resp)
            return resp.json()
        except Exception as e:
            last_err = e
            sleep_s = 2 * (attempt + 1)
            print(f"Search timeout or error (attempt {attempt+1}/5): {e}. Sleeping {sleep_s}s...")
            time.sleep(sleep_s)
    raise last_err


def collect_for_category(category: str, target_count: int):
    out_cat = OUT_DIR / category.replace("/", "_")
    out_cat.mkdir(parents=True, exist_ok=True)

    page = 1
    downloaded = 0
    seen_ids = set(p.stem for p in out_cat.glob("*.jpg"))
    print(f"\n=== Category: {category} (already: {len(seen_ids)}) ===")

    while downloaded < target_count:
        data = search_page(category, page)
        results = data.get("results", [])
        if not results:
            print(f"No more results at page {page} for '{category}'.")
            break

        for item in results:
            try:
                img_id = item["id"]
                created_at = item.get("created_at", "")
                if created_at <= CREATED_AFTER_ISO:
                    continue
                if img_id in seen_ids:
                    continue

                # Prefer 'urls'["regular"] or 'full' with reasonable size
                urls = item.get("urls", {})
                url = urls.get("regular") or urls.get("small") or urls.get("full")
                if not url:
                    continue

                out_path = out_cat / f"{img_id}.jpg"
                img_resp = requests.get(url, timeout=60)
                if img_resp.status_code == 429:
                    ratelimit_sleep(img_resp)
                    img_resp = requests.get(url, timeout=60)
                img_resp.raise_for_status()
                with open(out_path, "wb") as f:
                    f.write(img_resp.content)

                downloaded += 1
                seen_ids.add(img_id)
                if downloaded % 10 == 0 or downloaded == target_count:
                    print(f"{category}: {downloaded}/{target_count}")

                if downloaded >= target_count:
                    break
            except Exception as e:
                print(f"Skip one due to error: {e}")
                continue

        page += 1
        # Be polite between pages
        time.sleep(1)


if __name__ == "__main__":
    print(f"Output dir: {OUT_DIR.resolve()}")
    OUT_DIR.mkdir(parents=True, exist_ok=True)

    for cat in CATEGORIES:
        collect_for_category(cat, IMAGES_PER_CATEGORY)
    print("Done.")
