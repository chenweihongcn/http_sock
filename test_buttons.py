#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""用 Playwright 测试管理台按钮功能，捕获所有控制台日志和网络错误"""

import asyncio, json, time, sys
from playwright.async_api import async_playwright

URL    = "http://192.168.50.94:8088/"
USER   = "admin"
PASSWD = "Admin2026Strong9X"

async def run():
    failures = []
    console_msgs = []

    async with async_playwright() as pw:
        browser = await pw.chromium.launch(headless=True, args=["--no-sandbox"])
        ctx = await browser.new_context()
        page = await ctx.new_page()

        # 收集控制台消息
        def on_console(msg):
            txt = f"[{msg.type}] {msg.text}"
            console_msgs.append(txt)
            if msg.type in ("error", "warning"): print("  CONSOLE:", txt)

        page.on("console", on_console)

        # 收集页面错误
        page.on("pageerror", lambda e: failures.append(f"PAGE ERROR: {e}"))

        print("=" * 60)
        print("1. 登陆")
        print("=" * 60)

        await page.goto(URL)
        await page.wait_for_load_state("networkidle")

        # 找到登陆表单，检查是否已在主页还是在登陆页
        title = await page.title()
        print(f"   页面标题: {title}")
        print(f"   当前 URL: {page.url}")

        # 若有登陆表单（loginButton 是登陆页专属 ID）
        login_btn = page.locator("#loginButton")
        if await login_btn.count() > 0 and await login_btn.is_visible():
            await page.locator('input[type="text"]').first.fill(USER)
            await page.locator('input[type="password"]').first.fill(PASSWD)
            await login_btn.click()
            await page.wait_for_load_state("networkidle")
            print(f"   登陆后 URL: {page.url}")
        else:
            print("   已在主页或无登陆表单")

        # 截图：登陆后
        await page.screenshot(path="shot_01_after_login.png", full_page=True)

        # 检查是否看到用户列表
        user_rows = page.locator("table tbody tr, .user-row")
        count = await user_rows.count()
        print(f"   发现用户行: {count}")

        # ─── 测试「用户」tab 按钮 ─────────────────────────────────
        print("\n" + "=" * 60)
        print("2. 测试「用户」tab 各按钮")
        print("=" * 60)

        # 确保在「用户」tab
        user_tab = page.locator("button, a, .tab").filter(has_text="用户").first
        if await user_tab.is_visible():
            await user_tab.click()
            await page.wait_for_timeout(500)

        # 找第一个用户行进行测试
        rows = page.locator("table tbody tr")
        row_count = await rows.count()
        print(f"   用户表格行数: {row_count}")

        if row_count > 0:
            first_row = rows.nth(0)
            row_text = await first_row.inner_text()
            print(f"   第一行: {row_text[:60].strip()}")

            # 测试「禁用/启用」按钮
            btn_disable = first_row.locator("button").filter(has_text="禁用|启用")
            if await btn_disable.count() > 0:
                btn_text = await btn_disable.first.inner_text()
                print(f"\n   [测试] 按「{btn_text}」按钮...")
                await btn_disable.first.click()
                await page.wait_for_timeout(1500)
                # 检查 toast 是否出现
                toast = page.locator("#toastArea .toast")
                toast_visible = await toast.count() > 0
                toast_text = await toast.first.inner_text() if toast_visible else "(无)"
                print(f"   → Toast 可见: {toast_visible}, 内容: {toast_text}")
                if not toast_visible: failures.append("禁用按钮：no toast appeared")
                await page.screenshot(path="shot_02_after_disable.png")
            else:
                print("   ⚠️  未找到「禁用/启用」按钮")
                failures.append("未找到禁用/启用按钮")

            # 等 toast 消失
            await page.wait_for_timeout(2000)

            # 测试「设备」按钮
            btn_device = first_row.locator("button").filter(has_text="设备")
            if await btn_device.count() > 0:
                print(f"\n   [测试] 按「设备」按钮...")
                await btn_device.first.click()
                await page.wait_for_timeout(1000)
                modal = page.locator(".modal-overlay, .modal-box")
                modal_visible = await modal.count() > 0
                modal_text = await modal.first.inner_text() if modal_visible else "(无)"
                print(f"   → Modal 可见: {modal_visible}, 内容: {modal_text[:80]}")
                if not modal_visible:
                    toast = page.locator("#toastArea .toast")
                    toast_visible = await toast.count() > 0
                    print(f"   → Toast 可见: {toast_visible}")
                    if not toast_visible: failures.append("设备按钮：no modal and no toast appeared")
                await page.screenshot(path="shot_03_devices_modal.png")
                # 按取消/关闭
                cancel = page.locator(".modal-box button").filter(has_text="取消|关闭|Cancel")
                if await cancel.count() > 0:
                    await cancel.first.click()
                    await page.wait_for_timeout(500)
            else:
                print("   ⚠️  未找到「设备」按钮")
                failures.append("未找到设备按钮")

            # 测试「改密码」按钮
            btn_pwd = first_row.locator("button").filter(has_text="改密码")
            if await btn_pwd.count() > 0:
                print(f"\n   [测试] 按「改密码」按钮...")
                await btn_pwd.first.click()
                await page.wait_for_timeout(1000)
                modal = page.locator(".modal-overlay, .modal-box")
                modal_visible = await modal.count() > 0
                modal_text = await modal.first.inner_text() if modal_visible else "(无)"
                print(f"   → Modal 可见: {modal_visible}, 内容: {modal_text[:80]}")
                if not modal_visible: failures.append("改密码按钮：no modal appeared")
                await page.screenshot(path="shot_04_pwd_modal.png")
                cancel = page.locator(".modal-box button").filter(has_text="取消|关闭|Cancel")
                if await cancel.count() > 0:
                    await cancel.first.click()
                    await page.wait_for_timeout(500)
            else:
                print("   ⚠️  未找到「改密码」按钮")
                failures.append("未找到改密码按钮")

        # ─── 探测 DOM 结构 ────────────────────────────────────────
        print("\n" + "=" * 60)
        print("3. DOM 结构诊断")
        print("=" * 60)

        html_snippet = await page.eval_on_selector("body", "el => el.innerHTML.substring(0, 3000)")
        print("   body HTML (前3000字符):")
        print(html_snippet[:3000])

        # 检查 toastArea 是否存在
        toast_area = await page.locator("#toastArea").count()
        print(f"\n   #toastArea 存在: {toast_area > 0}")

        # 检查 JS 函数是否注册
        funcs = ["toast", "modalConfirm", "modalPrompt", "modalPassword",
                 "toggleUser", "deleteUser", "resetUserPassword"]
        for fn in funcs:
            defined = await page.evaluate(f"typeof {fn} === 'function'")
            print(f"   window.{fn}: {'✓' if defined else '✗ 未定义'}")
            if not defined: failures.append(f"JS 函数未定义: {fn}")

        # 检查按钮上绑定的 onclick
        btn_sample = await page.evaluate("""
            () => {
                const btns = document.querySelectorAll('button');
                return Array.from(btns).slice(0, 15).map(b => ({
                    text: b.innerText.trim(),
                    onclick: b.getAttribute('onclick') || b.onclick?.toString()?.slice(0,80) || null
                }));
            }
        """)
        print("\n   前15个按钮及其 onclick:")
        for b in btn_sample:
            print(f"     [{b['text'][:12]:<12}] onclick={b['onclick']}")

        # ─── 网络请求监控 ─────────────────────────────────────────
        print("\n" + "=" * 60)
        print("4. 最后截图")
        print("=" * 60)
        await page.screenshot(path="shot_05_final.png", full_page=True)
        print("   已保存 shot_05_final.png")

        await browser.close()

    # ─── 结果摘要 ─────────────────────────────────────────────────
    print("\n" + "=" * 60)
    print("测试结果摘要")
    print("=" * 60)
    if failures:
        print(f"✗ {len(failures)} 个问题:")
        for f in failures:
            print(f"  - {f}")
    else:
        print("✓ 所有测试通过")

    print("\n控制台消息:")
    for m in console_msgs[-20:]:  # 最后20条
        print(f"  {m}")

asyncio.run(run())
