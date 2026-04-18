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

        print("=== 第三阶段入口检查 ===")
        for sel, name in [
            ("#expiredCount", "已过期统计"),
            ("#expiring7Count", "7天内到期统计"),
            ("#nearQuotaCount", "流量告警统计"),
            ("#healthList", "巡检列表"),
            ("#refreshHealthBtn", "巡检按钮"),
            ("#filterExpiredBtn", "已过期快捷筛选"),
            ("#filterExpiring7Btn", "7天内快捷筛选"),
            ("#filterPermanentBtn", "永久快捷筛选"),
        ]:
            ok = await page.locator(sel).count() > 0
            print(f"{name}: {'OK' if ok else 'MISSING'}")

        print("\n=== 巡检面板数据检查 ===")
        expired = await page.inner_text("#expiredCount")
        exp7 = await page.inner_text("#expiring7Count")
        near = await page.inner_text("#nearQuotaCount")
        health_items = await page.locator("#healthList li").count()
        last_scan = await page.inner_text("#healthLastScan")
        print("已过期:", expired)
        print("7天内:", exp7)
        print("流量告警:", near)
        print("巡检项数量:", health_items)
        print("最近巡检:", last_scan)

        print("\n=== 快捷筛选行为检查 ===")
        await page.click("#filterPermanentBtn")
        await page.wait_for_timeout(700)
        expiry_filter = await page.input_value("#userExpiryFilter")
        active_tab = await page.locator(".tab.active").first.inner_text()
        print("筛选值:", expiry_filter)
        print("当前tab:", active_tab)

        print("\n=== 手动巡检按钮检查 ===")
        await page.click("#refreshHealthBtn")
        await page.wait_for_timeout(700)
        health_items2 = await page.locator("#healthList li").count()
        print("巡检后巡检项数量:", health_items2)

        print("\n页面错误数:", len(errors))
        if errors:
            for e in errors[:3]:
                print(e)

        await page.screenshot(path="shot_phase3_features.png", full_page=True)
        await browser.close()

asyncio.run(run())
