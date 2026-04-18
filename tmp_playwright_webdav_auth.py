import asyncio
import base64
import json
from playwright.async_api import async_playwright

BASE = "http://192.168.50.94:8088"
USER = "kangpu"
PWD = "ckp123456"

async def main():
    auth = base64.b64encode(f"{USER}:{PWD}".encode()).decode()
    async with async_playwright() as p:
        req = await p.request.new_context(base_url=BASE, extra_http_headers={"Authorization": f"Basic {auth}"})
        r = await req.get("/webdav/")
        body = await r.text()
        print(json.dumps({
            "status": r.status,
            "www_authenticate": r.headers.get("www-authenticate"),
            "body_prefix": body[:180]
        }, ensure_ascii=False, indent=2))
        await req.dispose()

asyncio.run(main())
