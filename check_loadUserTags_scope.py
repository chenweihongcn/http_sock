import asyncio
from playwright.async_api import async_playwright

async def main():
    async with async_playwright() as pw:
        browser = await pw.chromium.launch(headless=True)
        page = await browser.new_page()
        await page.goto('http://192.168.50.94:8088/', wait_until='domcontentloaded')
        await page.wait_for_timeout(1000)
        t1 = await page.evaluate("typeof loadUserTags")
        t2 = await page.evaluate("typeof bootstrapApp")
        t3 = await page.evaluate("typeof login")
        print('loadUserTags=', t1, 'bootstrapApp=', t2, 'login=', t3)
        await browser.close()

asyncio.run(main())
