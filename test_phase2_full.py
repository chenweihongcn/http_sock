#!/usr/bin/env python
# -*- coding: utf-8 -*-

import asyncio
from playwright.async_api import async_playwright

URL = "http://192.168.50.94:8088/"
USER = "admin"
PASSWD = "Admin2026Strong9X"

async def run():
    async with async_playwright() as pw:
        browser = await pw.chromium.launch(headless=True, args=["--no-sandbox"])
        page = await browser.new_page()

        errors = []
        page.on("pageerror", lambda e: errors.append(str(e)))

        await page.goto(URL, wait_until="domcontentloaded")
        await page.fill("#loginUsername", USER)
        await page.fill("#loginPassword", PASSWD)
        await page.click("#loginButton")
        await page.wait_for_function("!document.getElementById('appView').classList.contains('hidden')", timeout=8000)
        await page.wait_for_selector("table tbody tr", timeout=8000)

        print("=== 入口检查 ===")
        for sel, name in [
            ("#userTagFilter", "标签筛选"),
            ("#userExpiryFilter", "到期筛选"),
            ("#batchTagBtn", "批量打标"),
            ("#batchExtendBtn", "批量延期"),
            ("#batchTopupBtn", "批量充值"),
            ("#exportUsersBtn", "导出CSV"),
            ("#importUsersBtn", "导入CSV"),
        ]:
            ok = await page.locator(sel).count() > 0
            print(f"{name}: {'OK' if ok else 'MISSING'}")

        row = page.locator("table tbody tr").first
        print("\n=== 单用户标签检查 ===")
        await row.locator("button").filter(has_text="标签").first.click()
        await page.wait_for_timeout(500)
        modal = await page.locator(".modal-overlay").count()
        print("标签 modal:", "OK" if modal else "MISSING")
        if modal:
            await page.locator(".modal-box button").filter(has_text="取消").first.click()

        print("\n=== 筛选请求检查 ===")
        req_urls = []
        page.on("request", lambda r: req_urls.append(r.url) if "/api/admin/users?" in r.url else None)
        await page.select_option("#userTagFilter", "")
        await page.select_option("#userExpiryFilter", "permanent")
        await page.wait_for_timeout(800)
        has_expire = any("expire_filter=permanent" in u for u in req_urls)
        print("携带 expire_filter=permanent:", "YES" if has_expire else "NO")

        print("\n=== 批量打标弹窗检查 ===")
        check = page.locator(".user-check").first
        await check.check()
        await page.click("#batchTagBtn")
        await page.wait_for_timeout(500)
        modal2 = await page.locator(".modal-overlay").count()
        print("批量打标 modal:", "OK" if modal2 else "MISSING")
        if modal2:
            await page.locator(".modal-box button").filter(has_text="取消").first.click()

        print("\n=== 导出CSV检查 ===")
        async with page.expect_download() as dl_info:
            await page.click("#exportUsersBtn")
        download = await dl_info.value
        print("导出文件:", download.suggested_filename)

        print("\n页面错误数:", len(errors))
        if errors:
            for e in errors[:3]:
                print(e)

        await page.screenshot(path="shot_phase2_full.png", full_page=True)
        await browser.close()

asyncio.run(run())
