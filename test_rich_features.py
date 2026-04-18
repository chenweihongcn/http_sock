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

        events = []
        page.on("pageerror", lambda e: events.append(f"PAGEERROR: {e}"))

        await page.goto(URL, wait_until="domcontentloaded")
        await page.fill("#loginUsername", USER)
        await page.fill("#loginPassword", PASSWD)
        await page.click("#loginButton")
        await page.wait_for_function("!document.getElementById('appView').classList.contains('hidden')", timeout=8000)

        await page.wait_for_selector("table tbody tr", timeout=8000)
        row = page.locator("table tbody tr").first

        print("新增按钮可见性:")
        for text in ["延期", "充值", "审计"]:
            cnt = await row.locator("button").filter(has_text=text).count()
            print(f"  {text}: {'OK' if cnt > 0 else 'MISSING'}")

        # 延期：打开 modal 后取消
        await row.locator("button").filter(has_text="延期").first.click()
        await page.wait_for_timeout(500)
        modal_days = await page.locator(".modal-overlay").count()
        print("延期 modal:", "OK" if modal_days else "MISSING")
        cancel = page.locator(".modal-box button").filter(has_text="取消").first
        if await cancel.count() > 0:
            await cancel.click()

        # 充值：打开 modal 后取消
        await row.locator("button").filter(has_text="充值").first.click()
        await page.wait_for_timeout(500)
        modal_quota = await page.locator(".modal-overlay").count()
        print("充值 modal:", "OK" if modal_quota else "MISSING")
        cancel = page.locator(".modal-box button").filter(has_text="取消").first
        if await cancel.count() > 0:
            await cancel.click()

        # 审计：应切换到审计 tab 并填充 target
        await row.locator("button").filter(has_text="审计").first.click()
        await page.wait_for_timeout(800)
        active_tab = await page.locator(".tab.active").first.inner_text()
        target = await page.input_value("#auditTarget")
        print("审计跳转:", active_tab, "target=", target)

        # 批量删除按钮存在
        del_btn = await page.locator("#batchDeleteBtn").count()
        print("批量删除入口:", "OK" if del_btn else "MISSING")

        print("错误数:", len(events))
        for e in events[:5]:
            print(" ", e)

        await page.screenshot(path="shot_rich_features.png", full_page=True)
        await browser.close()

asyncio.run(run())
