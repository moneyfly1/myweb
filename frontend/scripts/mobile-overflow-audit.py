#!/usr/bin/env python3
"""移动端溢出审计（全站逐页几何测量）。

用法：
    pip install playwright && playwright install chromium
    export DSH_PROBE_TOKENS='{"admin":"<管理员 JWT>","user":"<普通用户 JWT>"}'
    python3 frontend/scripts/mobile-overflow-audit.py 390        # 也可 360 / 320

只做只读访问（打开页面 + 量尺寸），不点击任何按钮、不提交表单。

判定口径（避免误报）：
  * 越界     = 元素右边缘超出视口 → 真问题
  * 内部横滚 = 元素 overflow-x 为 auto/scroll 且内容更宽 → 需要横滑，能修就修
  * overflow:hidden + text-overflow:ellipsis 是**有意截断**，不算问题
  * input/textarea 内部文本滚动属正常行为，不算问题
"""
import json, os, sys
from playwright.sync_api import sync_playwright

BASE = os.environ.get("DSH_PROBE_BASE", "https://dy.moneyfly.top")
TOK = json.loads(os.environ.get("DSH_PROBE_TOKENS") or "{}")  # {"admin": "<jwt>", "user": "<jwt>"}
ADMIN_USER = {"id": 1, "username": "admin", "email": "admin@moneyfly.top", "is_admin": True, "balance": 0}
NORMAL_USER = {"id": 2, "username": "probe_user", "email": "user2@moneyfly.top", "is_admin": False, "balance": 0}

ROUTES = [
    ("user", "/dashboard"), ("user", "/subscription"), ("user", "/devices"),
    ("user", "/packages"), ("user", "/orders"), ("user", "/nodes"), ("user", "/help"),
    ("user", "/profile"), ("user", "/login-history"), ("user", "/tickets"),
    ("user", "/settings"), ("user", "/tutorials"), ("user", "/invites"),
    ("user", "/knowledge"),
    ("admin", "/admin/dashboard"), ("admin", "/admin/users"), ("admin", "/admin/abnormal-users"),
    ("admin", "/admin/config-update"), ("admin", "/admin/nodes"), ("admin", "/admin/custom-nodes"),
    ("admin", "/admin/selfhost-nodes"), ("admin", "/admin/subscriptions"), ("admin", "/admin/orders"),
    ("admin", "/admin/packages"), ("admin", "/admin/payment-config"), ("admin", "/admin/settings"),
    ("admin", "/admin/config"), ("admin", "/admin/statistics"), ("admin", "/admin/email-queue"),
    ("admin", "/admin/profile"), ("admin", "/admin/logs"), ("admin", "/admin/system-logs"),
    ("admin", "/admin/coupons"), ("admin", "/admin/tickets"), ("admin", "/admin/invites"),
    ("admin", "/admin/user-levels"), ("admin", "/admin/knowledge"), ("admin", "/admin/analytics"),
    ("admin", "/admin/promotions"),
]

JS = """(vw) => {
  const inDrawer = el => el.closest('.mobile-menu, .sidebar, .sidebar-nav, .el-drawer, [class*="mobile-menu"], .v-modal') !== null;
  const hidden = el => { const cs=getComputedStyle(el); return cs.display==='none'||cs.visibility==='hidden'||parseFloat(cs.opacity)===0; };
  const right=[], scroll=[];
  document.querySelectorAll('body *').forEach(el => {
    if (hidden(el) || inDrawer(el)) return;
    const r = el.getBoundingClientRect();
    if (r.width===0 && r.height===0) return;
    const tag = el.tagName.toLowerCase();
    const cls = (typeof el.className==='string'?el.className:'').split(' ').filter(Boolean).slice(0,3).join('.');
    const id = el.id ? '#'+el.id : '';
    if (r.right > vw + 1.5) right.push({sel: tag+id+(cls?'.'+cls:''), over: Math.round(r.right-vw), w: Math.round(r.width), t:(el.textContent||'').trim().replace(/\\s+/g,' ').slice(0,32)});
    const ox = getComputedStyle(el).overflowX;
    if ((ox === 'auto' || ox === 'scroll') && el.scrollWidth-el.clientWidth > 2 && el.clientWidth > 24 && !['INPUT','TEXTAREA','SELECT'].includes(tag))
      scroll.push({sel: tag+id+(cls?'.'+cls:''), d: el.scrollWidth-el.clientWidth, w: el.clientWidth, ox, t:(el.textContent||'').trim().replace(/\\s+/g,' ').slice(0,32)});
  });
  const txt = document.body.innerText;
  const raw = (txt.match(/\\{\\s*(item|row|scope|field|user|node)\\.[a-zA-Z_]+/g)||[]).slice(0,3);
  return {nRight: right.length, right: right.slice(0,8), nScroll: scroll.length, scroll: scroll.slice(0,8), raw,
          docW: document.documentElement.scrollWidth, vw: document.documentElement.clientWidth,
          title: (document.querySelector('h1')||document.querySelector('.page-title')||{textContent:''}).textContent.trim().slice(0,16),
          len: txt.length,
          empty: /暂无数据|暂无记录|没有数据/.test(txt)};
}"""

WIDTH = int(sys.argv[1]) if len(sys.argv) > 1 else 390
only = sys.argv[2] if len(sys.argv) > 2 else None

with sync_playwright() as p:
    b = p.chromium.launch()
    ctx = b.new_context(viewport={"width": WIDTH, "height": 844}, is_mobile=True, has_touch=True,
                        device_scale_factor=2, ignore_https_errors=True)
    ctx.add_init_script("""
        const T = %s, AU = %s, UU = %s;
        const put = (k,v) => localStorage.setItem(k, JSON.stringify({value:v, expiry:4102444800000, timestamp:1}));
        put('cboard_secure_admin_token', T.admin); put('cboard_secure_admin_user', AU);
        put('cboard_secure_user_token', T.user);   put('cboard_secure_user_data', UU);
        put('cboard_secure_user_remember', true);
    """ % (json.dumps(TOK), json.dumps(ADMIN_USER), json.dumps(NORMAL_USER)))
    pg = ctx.new_page()
    errs = []
    pg.on("pageerror", lambda e: errs.append(str(e)[:110]))
    pg.on("console", lambda m: errs.append("c:"+m.text[:110]) if m.type == "error" else None)

    results = []
    for role, path in ROUTES:
        if only and only not in path: continue
        errs.clear()
        try:
            pg.goto(BASE + path, wait_until="domcontentloaded", timeout=45000)
            try: pg.wait_for_load_state("networkidle", timeout=9000)
            except Exception: pass
            pg.wait_for_timeout(1800)
            r = pg.evaluate(JS, WIDTH)
            results.append((path, r, list(errs)))
            flag = "❗" if (r["nRight"] or r["raw"] or errs) else ("⚠️" if r["nScroll"] else "✅")
            print(f"{flag} {path:28} {r['title']:10} 越界{r['nRight']:3} 横滚{r['nScroll']:3} 错误{len(errs):2} 文本{r['len']:6} 文档{r['docW']}/{r['vw']}" + (" [空数据]" if r["empty"] else ""))
            for x in r["right"][:6]: print(f"      越界 {x['sel'][:60]:60} 超{x['over']:4}px w={x['w']:4} | {x['t']}")
            for x in r["scroll"][:6]: print(f"      横滚 {x['sel'][:60]:60} 溢出{x['d']:4}px w={x['w']:4} | {x['t']}")
            if r["raw"]: print(f"      ❗原始模板: {r['raw']}")
            for e in errs[:2]: print(f"      ❗错误: {e}")
        except Exception as e:
            print(f"?? {path} 探测失败: {str(e)[:80]}")
    json.dump([{"path": p_, **r, "errs": e} for p_, r, e in results], open(f'/tmp/mfcheck/sweep_final_{WIDTH}.json','w'), ensure_ascii=False)
    print(f"\n汇总：{len(results)} 个页面 → 越界 {sum(1 for _,r,_ in results if r['nRight'])} 个，横滚 {sum(1 for _,r,_ in results if r['nScroll'])} 个，报错 {sum(1 for _,_,e in results if e)} 个")
    b.close()
