import asyncio
import json
from playwright.async_api import async_playwright

BASE = "http://192.168.50.94:8088"
ADMIN_USER = "admin"
ADMIN_PASS = "admin123"
TARGET_USER = "kangpu"

async def main():
    result = {
        "login": None,
        "kangpu_exists": None,
        "kangpu_smb_enabled": None,
        "kangpu_smb_path": None,
        "webdav_status": None,
        "webdav_www_authenticate": None,
        "errors": []
    }

    async with async_playwright() as p:
        req = await p.request.new_context(base_url=BASE)

        # 1) admin login
        try:
            r = await req.post("/api/admin/login", data={"username": ADMIN_USER, "password": ADMIN_PASS})
            result["login"] = r.status
            if r.status != 200:
                result["errors"].append(f"admin login failed status={r.status} body={await r.text()}")
            else:
                body = await r.json()
                csrf = body.get("csrf_token", "")

                # 2) fetch users and inspect kangpu
                ru = await req.get("/api/admin/users?offset=0&limit=200")
                if ru.status == 200:
                    users = (await ru.json()).get("items", [])
                    target = next((u for u in users if u.get("username") == TARGET_USER), None)
                    result["kangpu_exists"] = target is not None
                    if target is not None:
                        result["kangpu_smb_enabled"] = target.get("smb_enabled")

                    # 3) fetch smb-path endpoint for kangpu (requires auth cookie from same context)
                    rp = await req.get(f"/api/admin/users/{TARGET_USER}/smb-path")
                    if rp.status == 200:
                        pbody = await rp.json()
                        result["kangpu_smb_path"] = pbody.get("smb_path")
                    else:
                        result["errors"].append(f"smb-path failed status={rp.status} body={await rp.text()}")
                else:
                    result["errors"].append(f"list users failed status={ru.status} body={await ru.text()}")

                # 4) webdav endpoint reachability (without auth should be 401 + WWW-Authenticate)
                rw = await req.get("/webdav/")
                result["webdav_status"] = rw.status
                result["webdav_www_authenticate"] = rw.headers.get("www-authenticate")
        except Exception as e:
            result["errors"].append(str(e))

        await req.dispose()

    print(json.dumps(result, ensure_ascii=False, indent=2))

asyncio.run(main())
