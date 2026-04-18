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

        print("=== UI入口检查 ===")
        checks = {
            "#userExpiryFilter": "到期筛选",
            "#batchExtendBtn": "批量延期",
            "#batchTopupBtn": "批量充值",
            "#exportUsersBtn": "导出CSV",
            "#importUsersBtn": "导入CSV",
            "#batchDeleteBtn": "批量删除",
        }
        for sel, name in checks.items():
            ok = await page.locator(sel).count() > 0
            print(f"{name}: {'OK' if ok else 'MISSING'}")

        print("\n=== 到期筛选检查 ===")
        await page.select_option("#userExpiryFilter", "permanent")
        await page.wait_for_timeout(800)
        rows = page.locator("table tbody tr")
        row_count = await rows.count()
        print("permanent 行数:", row_count)

        await page.select_option("#userExpiryFilter", "expired")
        await page.wait_for_timeout(800)
        row_count_exp = await page.locator("table tbody tr").count()
        print("expired 行数:", row_count_exp)

        await page.select_option("#userExpiryFilter", "")
        await page.wait_for_timeout(800)

        print("\n=== 批量延期/充值弹窗检查 ===")
        # 勾选第一行
        first_check = page.locator(".user-check").first
        await first_check.check()
        await page.wait_for_timeout(300)

        await page.click("#batchExtendBtn")
        await page.wait_for_timeout(500)
        modal1 = await page.locator(".modal-overlay").count()
        print("批量延期 modal:", "OK" if modal1 else "MISSING")
        if modal1:
            await page.locator(".modal-box button").filter(has_text="取消").first.click()

        await page.click("#batchTopupBtn")
        await page.wait_for_timeout(500)
        modal2 = await page.locator(".modal-overlay").count()
        print("批量充值 modal:", "OK" if modal2 else "MISSING")
        if modal2:
            await page.locator(".modal-box button").filter(has_text="取消").first.click()

        print("\n=== 导出CSV检查 ===")
        async with page.expect_download() as dl_info:
            await page.click("#exportUsersBtn")
        download = await dl_info.value
        print("导出文件:", download.suggested_filename)

        print("\n页面错误数:", len(errors))
        if errors:
            print(errors[:3])

        await page.screenshot(path="shot_phase2_features.png", full_page=True)
        await browser.close()

asyncio.run(run())
