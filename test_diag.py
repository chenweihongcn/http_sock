#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""深度诊断：拦截所有网络请求，捕获 JS 错误，判断按钮点击是否到达后端"""

import asyncio, json
from playwright.async_api import async_playwright

URL    = "http://192.168.50.94:8088/"
USER   = "admin"
PASSWD = "Admin2026Strong9X"

async def run():
    async with async_playwright() as pw:
        browser = await pw.chromium.launch(headless=True, args=["--no-sandbox"])
        ctx = await browser.new_context()
        page = await ctx.new_page()

        requests_log = []
        console_log  = []

        # 拦截所有网络请求
        def on_request(req):
            if "/api/" in req.url:
                requests_log.append(f"→ {req.method} {req.url}")

        async def on_response(res):
            if "/api/" in res.url:
                try:
                    body = await res.text()
                except:
                    body = "(unreadable)"
                requests_log.append(f"← {res.status} {res.url} | {body[:120]}")

        page.on("request",  on_request)
        page.on("response", on_response)
        page.on("console",  lambda m: console_log.append(f"[{m.type}] {m.text}"))
        page.on("pageerror",lambda e: console_log.append(f"[PAGEERROR] {e}"))

        print("="*60)
        print("步骤1: 加载首页")
        print("="*60)
        await page.goto(URL, wait_until="domcontentloaded")
        await page.wait_for_timeout(2000)

        # 打印页面加载期间的 API 请求
        print("页面加载 API 请求:")
        for r in requests_log:
            print(" ", r)
        requests_log.clear()

        print("\n" + "="*60)
        print("步骤2: 登陆")
        print("="*60)
        await page.fill("#loginUsername", USER)
        await page.fill("#loginPassword", PASSWD)
        print(f"  已填: {USER} / {PASSWD}")

        await page.click("#loginButton")
        print("  已点击[登录]")

        # 等待 loginStatus 出现内容（最多5秒）
        try:
            await page.wait_for_function(
                "document.getElementById('loginStatus').textContent.trim() !== ''",
                timeout=5000
            )
        except:
            pass

        # 再等 appView 可能出现（最多3秒）
        try:
            await page.wait_for_function(
                "!document.getElementById('appView').classList.contains('hidden')",
                timeout=3000
            )
        except:
            pass

        login_status = await page.eval_on_selector("#loginStatus", "el => el.textContent")
        app_visible  = await page.evaluate("!document.getElementById('appView').classList.contains('hidden')")
        print(f"  loginStatus: {login_status!r}")
        print(f"  appView 可见: {app_visible}")

        print("\n  登陆期间 API 请求:")
        for r in requests_log:
            print("   ", r)
        requests_log.clear()

        await page.screenshot(path="diag_01_after_login.png", full_page=True)

        if not app_visible:
            print("\n  ⚠️  登陆失败，停止测试")
            print("\n控制台日志:")
            for m in console_log: print(" ", m)
            await browser.close()
            return

        print("\n" + "="*60)
        print("步骤3: 等待用户列表加载")
        print("="*60)
        try:
            await page.wait_for_selector("table tbody tr", timeout=5000)
        except:
            print("  ⚠️  5s 内未出现用户行")

        rows = page.locator("table tbody tr")
        cnt  = await rows.count()
        print(f"  用户行数: {cnt}")
        for i in range(min(cnt, 3)):
            txt = await rows.nth(i).inner_text()
            print(f"  row[{i}]: {txt[:80]}")

        requests_log.clear()
        console_log.clear()

        if cnt == 0:
            print("  ⚠️  无用户行，无法测试按钮")
            await browser.close()
            return

        print("\n" + "="*60)
        print("步骤4: 测试第一行各按钮")
        print("="*60)
        first = rows.nth(0)

        async def click_btn(label):
            requests_log.clear()
            console_log.clear()
            btn = first.locator("button").filter(has_text=label)
            cnt2 = await btn.count()
            if cnt2 == 0:
                print(f"  [{label}] ⚠️  按钮不存在")
                return
            is_vis = await btn.first.is_visible()
            is_ena = await btn.first.is_enabled()
            print(f"  [{label}] 可见={is_vis} 可用={is_ena}")
            await btn.first.click()
            await page.wait_for_timeout(1500)

            # 检查 modal / toast
            modal_cnt = await page.locator(".modal-overlay").count()
            toast_cnt = await page.locator("#toastArea .toast").count()
            modal_txt = await page.locator(".modal-overlay").first.inner_text() if modal_cnt else ""
            toast_txt = await page.locator("#toastArea .toast").first.inner_text() if toast_cnt else ""
            print(f"    modal={modal_cnt} toast={toast_cnt}")
            if modal_txt: print(f"    modal内容: {modal_txt[:80]}")
            if toast_txt: print(f"    toast内容: {toast_txt[:80]}")

            # 打印后端请求
            if requests_log:
                print(f"    API请求: {requests_log}")
            else:
                print(f"    API请求: (无)")

            # console 错误
            errors = [m for m in console_log if "error" in m.lower()]
            if errors:
                print(f"    JS错误: {errors}")

            # 关掉 modal（如果有）
            cancel = page.locator(".modal-box button").filter(has_text="取消")
            if await cancel.count():
                await cancel.first.click()
                await page.wait_for_timeout(300)

        for label in ["禁用", "启用", "设备", "清流量", "改密码"]:
            await click_btn(label)
            print()

        print("="*60)
        print("步骤5: 控制台汇总")
        print("="*60)
        for m in console_log:
            print(" ", m)

        await page.screenshot(path="diag_02_final.png", full_page=True)
        await browser.close()

asyncio.run(run())
