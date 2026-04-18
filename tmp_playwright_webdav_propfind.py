import asyncio
import base64
import json
from playwright.async_api import async_playwright

BASE = "http://192.168.50.94:8088"
USER = "kangpu"
PWD = "ckp123456"

async def main():
    async with async_playwright() as p:
        # no auth
        req0 = await p.request.new_context(base_url=BASE)
        r0 = await req0.fetch('/webdav/', method='PROPFIND', headers={'Depth':'1'})
        t0 = await r0.text()
        await req0.dispose()

        # with auth
        auth = base64.b64encode(f"{USER}:{PWD}".encode()).decode()
        req1 = await p.request.new_context(base_url=BASE, extra_http_headers={'Authorization': f'Basic {auth}'})
        r1 = await req1.fetch('/webdav/', method='PROPFIND', headers={'Depth':'1'})
        t1 = await r1.text()
        await req1.dispose()

        print(json.dumps({
            'no_auth_status': r0.status,
            'no_auth_www_authenticate': r0.headers.get('www-authenticate'),
            'no_auth_body_prefix': t0[:120],
            'with_auth_status': r1.status,
            'with_auth_www_authenticate': r1.headers.get('www-authenticate'),
            'with_auth_body_prefix': t1[:200]
        }, ensure_ascii=False, indent=2))

asyncio.run(main())
