#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""捕获 JS 栈溢出的详细调用栈"""

import asyncio
from playwright.async_api import async_playwright

URL    = "http://192.168.50.94:8088/"
USER   = "admin"
PASSWD = "Admin2026Strong9X"

async def run():
    async with async_playwright() as pw:
        browser = await pw.chromium.launch(headless=True, args=["--no-sandbox"])
        ctx = await browser.new_context()
        page = await ctx.new_page()

        errors = []
        page.on("pageerror", lambda e: errors.append(str(e)))
        page.on("console",   lambda m: print(f"  [{m.type}] {m.text[:200]}") if m.type in ("error","warning") else None)

        await page.goto(URL, wait_until="domcontentloaded")
        await page.wait_for_timeout(1500)

        # 注入特殊版 toggleUser，打印调用前后的信息
        await page.evaluate("""
        () => {
            const origToggle = window.toggleUser;
            window.toggleUser = async function(username, action) {
                console.log('toggleUser called: ' + username + ' / ' + action);
                try {
                    await origToggle(username, action);
                    console.log('toggleUser done');
                } catch(e) {
                    console.error('toggleUser threw: ' + e.message + '\\n' + (e.stack||'no-stack').substring(0,800));
                }
            };
            
            const origToast = window.toast;
            window.toast = function(msg, type) {
                console.log('toast: ' + msg + ' [' + type + ']');
                return origToast ? origToast(msg, type) : undefined;
            };
        }
        """)

        # 登陆
        await page.fill("#loginUsername", USER)
        await page.fill("#loginPassword", PASSWD)
        await page.click("#loginButton")
        try:
            await page.wait_for_function("!document.getElementById('appView').classList.contains('hidden')", timeout=8000)
        except:
            print("登陆失败")
            await browser.close(); return

        print("登陆成功")
        await page.wait_for_timeout(1000)

        # 直接调用 toggleUser
        errors.clear()
        print("\n--- 直接调用 toggleUser('legalcoop','disable') ---")
        await page.evaluate("toggleUser('legalcoop', 'disable')")
        await page.wait_for_timeout(3000)

        print("\n--- 调用后 pageerror ---")
        for e in errors:
            print(e[:2000])

        # 检查 toast
        toast_cnt = await page.locator("#toastArea .toast").count()
        print(f"\ntoast 数量: {toast_cnt}")

        # 尝试直接调用 api()
        errors.clear()
        print("\n--- 直接调用 api POST disable ---")
        result = await page.evaluate("""
        async () => {
            try {
                const r = await fetch('/api/admin/users/legalcoop/disable', {
                    method: 'POST',
                    credentials: 'same-origin'
                });
                return {status: r.status, body: await r.text()};
            } catch(e) {
                return {error: e.message, stack: (e.stack||'').substring(0,500)};
            }
        }
        """)
        print("fetch 结果:", result)

        await browser.close()

asyncio.run(run())
