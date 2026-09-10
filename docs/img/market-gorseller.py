#!/usr/bin/env python3
"""JetBrains Marketplace için 1200x760 tanıtım görselleri üretir.

    python3 docs/img/market-gorseller.py

Marketplace ekran görüntüsü için minimum 1200x760 istiyor. Görseller headless
Chrome ile 2x çözünürlükte alınıp tam 1200x760'a indiriliyor (metin kenarları
1x render'da tırtıklı çıkıyor).

Çıktı: docs/img/market-<ad>.png (İngilizce) + market-<ad>.tr.png (Türkçe).

Renkler IntelliJ New UI koyu temasından; metin kontrastları WCAG AA'ya göre
seçildi (gövde metni >7:1, ikincil metin >4.5:1) — küçük puntoda okunması
gereken tanıtım görseli.
"""
import html
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

DIZIN = Path(__file__).resolve().parent
GEN = 1200, 760

# ── tema ─────────────────────────────────────────────────────────────────────
CSS = """
*{margin:0;padding:0;box-sizing:border-box}
:root{
  --bg:#1e1f22; --panel:#2b2d30; --panel2:#26282b; --line:#393b40;
  --txt:#dfe1e5;            /* #1e1f22 üzerinde 12.6:1 */
  --dim:#a7abb2;            /* #2b2d30 üzerinde 6.4:1  */
  --accent:#548af7; --key:#e8bd6b; --str:#6aab73; --kw:#cf8e6d;
  --think:#7fb2ff; --write:#69c98c; --sel:#214283;
}
body{width:1200px;height:760px;background:var(--bg);color:var(--txt);
  font:13px/1.5 "DejaVu Sans","Liberation Sans",sans-serif;overflow:hidden}
.win{display:flex;flex-direction:column;height:760px}
.title{height:34px;flex:0 0 34px;background:var(--panel);border-bottom:1px solid var(--line);
  display:flex;align-items:center;gap:10px;padding:0 12px;font-size:12px;color:var(--dim)}
.dot{width:11px;height:11px;border-radius:50%}
.crumb{color:var(--txt);font-weight:600}
.mid{flex:1;display:flex;min-height:0}
.tree{width:212px;flex:0 0 212px;background:var(--panel2);border-right:1px solid var(--line);
  padding:8px 0;font-size:12.5px}
.tree div{padding:3px 12px;white-space:nowrap;overflow:hidden}
.tree .on{background:var(--sel);color:#fff}
.tree .g{color:var(--dim)}
.ed{flex:1;min-width:0;display:flex;flex-direction:column}
.tabs{height:30px;flex:0 0 30px;display:flex;align-items:stretch;background:var(--panel2);
  border-bottom:1px solid var(--line);font-size:12px}
.tabs span{display:flex;align-items:center;padding:0 14px;color:var(--dim)}
.tabs span.on{color:var(--txt);background:var(--bg);border-bottom:2px solid var(--accent)}
pre{flex:1;padding:12px 0 0 0;font:13px/1.62 "DejaVu Sans Mono","Liberation Mono",monospace;font-style:normal;
  color:var(--txt);overflow:hidden}
pre i{display:block;padding:0 14px;font-style:normal}
pre i b{display:inline-block;width:26px;color:#5a5d63;font-weight:400;text-align:right;
  margin-right:14px;user-select:none}
.hl{background:var(--sel)}
.kw{color:var(--kw)} .st{color:var(--str)} .fn{color:#56a8f5} .cm{color:#8b9099}
.tw{flex:0 0 276px;background:var(--panel2);border-top:1px solid var(--line);
  display:flex;flex-direction:column}
.twbar{height:30px;flex:0 0 30px;display:flex;align-items:center;gap:14px;padding:0 12px;
  border-bottom:1px solid var(--line);font-size:12px;color:var(--dim)}
.twbar b{color:var(--txt)}
.twtab{padding:2px 10px;border-radius:5px;background:var(--bg);color:var(--txt)}
.out{flex:1;padding:10px 14px;font:12.5px/1.62 "DejaVu Sans Mono","Liberation Mono",monospace;
  overflow:hidden;white-space:pre-wrap}
.th{color:var(--think);font-weight:700} .wr{color:var(--write);font-weight:700}
.dm{color:var(--dim)} .ky{color:var(--key)}
.sep{color:#4a4d52}
.inp{flex:0 0 auto;border-top:1px solid var(--line);padding:9px 12px;display:flex;gap:9px;
  align-items:flex-start;background:var(--panel)}
.mode{flex:0 0 96px;background:var(--bg);border:1px solid var(--line);border-radius:6px;
  padding:6px 9px;font-size:12.5px;color:var(--txt);display:flex;justify-content:space-between}
.box{flex:1;background:var(--bg);border:1px solid var(--accent);border-radius:6px;
  padding:7px 10px;font:12.5px/1.55 "DejaVu Sans Mono","Liberation Mono",monospace;min-height:52px}
.cur{display:inline-block;width:7px;height:15px;background:var(--txt);vertical-align:-3px}
.hint{padding:2px 12px 8px;font-size:11.5px;color:var(--dim);background:var(--panel)}
.menu{position:absolute;left:470px;top:210px;width:340px;background:var(--panel);
  border:1px solid var(--line);border-radius:8px;padding:5px;font-size:12.5px;
  box-shadow:0 14px 34px rgba(0,0,0,.55)}
.menu div{display:flex;justify-content:space-between;padding:6px 10px;border-radius:5px}
.menu div.on{background:var(--sel);color:#fff}
.menu .sc{color:var(--dim)} .menu div.on .sc{color:#cfd8ea}
.menu hr{border:0;border-top:1px solid var(--line);margin:5px 8px}
.menu .hdr{color:var(--dim);font-size:11.5px;padding:5px 10px 7px}
.badge{position:absolute;right:22px;top:286px;background:var(--panel);border:1px solid var(--line);
  border-radius:8px;padding:9px 13px;font-size:12px;color:var(--dim);line-height:1.75}
.badge b{color:var(--txt)}
"""

def kod(satirlar, ilk=41):
    """Kod bloğu: (metin, seçili mi) çiftlerinden satır numaralı <pre> üretir."""
    out = []
    for i, (met, sec) in enumerate(satirlar):
        out.append(f'<i class="{"hl" if sec else ""}"><b>{ilk+i}</b>{met}</i>')
    return "".join(out)


def cerceve(govde, baslik, tree_on, menu="", badge=""):
    return f"""<div class="win">
  <div class="title">
    <span class="dot" style="background:#ff5f57"></span>
    <span class="dot" style="background:#febc2e"></span>
    <span class="dot" style="background:#28c840"></span>
    <span style="width:8px"></span>GoLand &nbsp;—&nbsp; <span class="crumb">cem</span>
    <span style="margin-left:auto">{html.escape(baslik)}</span>
  </div>
  <div class="mid">
    <div class="tree">
      <div class="g">cem</div>
      <div>&nbsp;&nbsp;config.go</div>
      <div class="{'on' if tree_on=='executor' else ''}">&nbsp;&nbsp;executor.go</div>
      <div>&nbsp;&nbsp;noise.go</div>
      <div>&nbsp;&nbsp;spinner.go</div>
      <div class="g">&nbsp;&nbsp;plugin/</div>
      <div>&nbsp;&nbsp;&nbsp;&nbsp;intellij/</div>
      <div class="g" style="margin-top:6px">.cem.yaml</div>
      <div>&nbsp;&nbsp;Makefile</div>
    </div>
    {govde}
  </div>
</div>{menu}{badge}"""


def sayfa(icerik):
    return f"<!doctype html><meta charset=utf-8><style>{CSS}</style><body>{icerik}</body>"


# ── metinler ────────────────────────────────────────────────────────────────
EN = dict(
    title="pair — thinker plans, writer codes",
    tabs=("executor.go", "config.go"),
    code=[("<span class=kw>func</span> (e *Executor) <span class=fn>runTool</span>(t Tool, in <span class=kw>string</span>) (<span class=kw>string</span>, <span class=kw>error</span>) {", False),
          ("\tcmd := exec.<span class=fn>Command</span>(t.Bin, t.Args...)", True),
          ("\tcmd.Stdin = strings.<span class=fn>NewReader</span>(in)", True),
          ("\tout, err := cmd.<span class=fn>Output</span>()", True),
          ("\t<span class=kw>if</span> err != <span class=kw>nil</span> {", True),
          ("\t\t<span class=kw>return</span> <span class=st>\"\"</span>, err", True),
          ("\t}", True),
          ("\t<span class=kw>return</span> <span class=fn>noiseFilter</span>(<span class=kw>string</span>(out)), <span class=kw>nil</span>", False),
          ("", False),
          ("<span class=cm>// noiseFilter strips the CLI banners around the answer.</span>", False),
          ("<span class=kw>func</span> <span class=fn>noiseFilter</span>(s <span class=kw>string</span>) <span class=kw>string</span> {", False),
          ("\t<span class=kw>var</span> b strings.Builder", False),
          ("\t<span class=kw>for</span> _, ln := <span class=kw>range</span> strings.<span class=fn>Split</span>(s, <span class=st>\"\\n\"</span>) {", False),
          ("\t\t<span class=kw>if</span> <span class=fn>isBanner</span>(ln) {", False),
          ("\t\t\t<span class=kw>continue</span>", False),
          ("\t\t}", False),
          ("\t\tb.<span class=fn>WriteString</span>(ln + <span class=st>\"\\n\"</span>)", False),
          ("\t}", False),
          ("\t<span class=kw>return</span> b.<span class=fn>String</span>()", False),
          ("}", False)],
    twtab="pair · executor.go",
    twbar="thinker claude · writer codex · high/low effort",
    out=[("th", "◆ thinker · claude · 12.4s"),
         ("dm", "  add retries with exponential backoff"),
         ("", "  • wrap the exec call in a retry loop, 4 attempts"),
         ("", "  • backoff 250ms · 500ms · 1s, ±20% jitter"),
         ("", "  • retry only on rate-limit and network errors"),
         ("", "  • keep the noiseFilter call on the final output"),
         ("sep", "  ────────────────────────────────────────────────"),
         ("wr", "◆ writer · codex · 7.8s"),
         ("", "  executor.go — runTool now retries (+34 −6)"),
         ("", "  executor_test.go — 3 cases, all passing"),
         ("dm", "  total 20.2s · plan 15 lines · cache: thinker HIT")],
    onceki=[("dm", "◆ cem 20260910.02 · pair · claude → codex"),
            ("", "  Type below and press Enter. The project root is the"),
            ("", "  working directory, so .cem.yaml applies as in the terminal."),
            ("sep", "  ────────────────────────────────────────────────"),
            ("th", "◆ thinker · claude · 9.6s"),
            ("", "  config.go — split ResolvedConfig, keep the YAML tags"),
            ("wr", "◆ writer · codex · 5.1s"),
            ("", "  config.go (+41 −12) · config_test.go 4 cases pass")],
    inp_mode="pair", inp_text="add retries with exponential backoff",
    hint="Enter sends · Shift+Enter new line · ↑/↓ earlier prompts · the project root is the working directory",
    badge="<b>No selection?</b><br>The cursor lands in the input box —<br>no modal dialog, no file is sent by mistake.",
    menu_hdr="cem — Compose · Execute · Multiplex",
    menu=[("cem: pair on selection", "Ctrl+Alt+P", True),
          ("cem: think on selection", "Ctrl+Alt+I", False),
          ("cem: write on selection", "Ctrl+Alt+W", False),
          ("cem: ask freely…", "Ctrl+Alt+A", False)],
    menu_badge="<b>Editor context menu</b><br>Also under Tools → cem, and on files<br>in the project tree (review / fix / explain).",
)

TR = dict(
    title="ikili — düşünen planlar, yazan kodlar",
    tabs=("executor.go", "config.go"),
    code=EN["code"],
    twtab="ikili · executor.go",
    twbar="düşünen claude · yazan codex · yüksek/düşük efor",
    out=[("th", "◆ düşünen · claude · 12.4s"),
         ("dm", "  üstel geri çekilmeli yeniden deneme ekle"),
         ("", "  • exec çağrısını 4 denemeli döngüye al"),
         ("", "  • bekleme 250ms · 500ms · 1s, ±%20 jitter"),
         ("", "  • yalnız hız sınırı ve ağ hatasında tekrar dene"),
         ("", "  • son çıktıda noiseFilter çağrısı kalsın"),
         ("sep", "  ────────────────────────────────────────────────"),
         ("wr", "◆ yazan · codex · 7.8s"),
         ("", "  executor.go — runTool yeniden deniyor (+34 −6)"),
         ("", "  executor_test.go — 3 durum, hepsi geçiyor"),
         ("dm", "  toplam 20.2s · plan 15 satır · önbellek: düşünen HIT")],
    onceki=[("dm", "◆ cem 20260910.02 · ikili · claude → codex"),
            ("", "  Aşağıya yaz ve Enter'a bas. Çalışma dizini proje kökü,"),
            ("", "  yani .cem.yaml terminaldeki gibi geçerli."),
            ("sep", "  ────────────────────────────────────────────────"),
            ("th", "◆ düşünen · claude · 9.6s"),
            ("", "  config.go — ResolvedConfig ayrılsın, YAML tag'leri kalsın"),
            ("wr", "◆ yazan · codex · 5.1s"),
            ("", "  config.go (+41 −12) · config_test.go 4 durum geçiyor")],
    inp_mode="ikili", inp_text="üstel geri çekilmeli yeniden deneme ekle",
    hint="Enter gönderir · Shift+Enter satır ekler · ↑/↓ önceki istemler · çalışma dizini proje kökü",
    badge="<b>Seçim yok mu?</b><br>İmleç giriş kutusuna gelir —<br>modal pencere yok, yanlışlıkla dosya gitmez.",
    menu_hdr="cem — Compose · Execute · Multiplex",
    menu=[("cem: seçimde ikili", "Ctrl+Alt+P", True),
          ("cem: seçimi düşün", "Ctrl+Alt+I", False),
          ("cem: seçimi yaz", "Ctrl+Alt+W", False),
          ("cem: serbest sor…", "Ctrl+Alt+A", False)],
    menu_badge="<b>Editör sağ-tık menüsü</b><br>Tools → cem altında ve proje ağacındaki<br>dosyalarda da var (incele / düzelt / açıkla).",
)


def editor(t, tw):
    return f"""<div class="ed">
      <div class="tabs"><span class="on">{t['tabs'][0]}</span><span>{t['tabs'][1]}</span></div>
      <pre>{kod(t['code'])}</pre>
      {tw}
    </div>"""


def cikti(t):
    satir = []
    for cls, met in t["out"]:
        c = {"th": "th", "wr": "wr", "dm": "dm", "sep": "sep"}.get(cls, "")
        satir.append(f'<span class="{c}">{html.escape(met)}</span>')
    return "\n".join(satir)


def g_pair(t):
    tw = f"""<div class="tw">
        <div class="twbar"><b>cem</b><span class="twtab">{html.escape(t['twtab'])}</span>
          <span>{html.escape(t['twbar'])}</span></div>
        <div class="out">{cikti(t)}</div>
      </div>"""
    return cerceve(editor(t, tw), t["title"], "executor")


def onceki(t):
    """input görselinde giriş kutusunun üstünde duran önceki koşu çıktısı."""
    return "\n".join(f'<span class="{c}">{html.escape(m)}</span>'
                     for c, m in t["onceki"])


def g_input(t):
    tw = f"""<div class="tw">
        <div class="twbar"><b>cem</b><span class="twtab">{html.escape(t['twtab'])}</span>
          <span>{html.escape(t['twbar'])}</span></div>
        <div class="out">{onceki(t)}</div>
        <div class="inp">
          <div class="mode"><span>{html.escape(t['inp_mode'])}</span><span class="dm">▾</span></div>
          <div class="box">{html.escape(t['inp_text'])}<span class="cur"></span></div>
        </div>
        <div class="hint">{html.escape(t['hint'])}</div>
      </div>"""
    return cerceve(editor(t, tw), t["title"], "executor",
                   badge=f'<div class="badge">{t["badge"]}</div>')


def g_menu(t):
    ogeler = "".join(
        f'<div class="{"on" if on else ""}"><span>{html.escape(ad)}</span>'
        f'<span class="sc">{html.escape(kis)}</span></div>'
        for ad, kis, on in t["menu"])
    tw = f"""<div class="tw" style="flex:0 0 110px">
        <div class="twbar"><b>cem</b><span class="twtab">{html.escape(t['twtab'])}</span></div>
        <div class="out"><span class="dm">{html.escape(t['hint'])}</span></div>
      </div>"""
    menu = f"""<div class="menu"><div class="hdr">{html.escape(t['menu_hdr'])}</div>
      {ogeler}<hr><div><span>cem: review file…</span><span class="sc"></span></div></div>"""
    return cerceve(editor(t, tw), t["title"], "executor", menu=menu,
                   badge=f'<div class="badge">{t["menu_badge"]}</div>')


GORSELLER = {"pair": g_pair, "input": g_input, "menu": g_menu}


def chrome_bul():
    for c in ("google-chrome", "chromium", "chromium-browser"):
        if shutil.which(c):
            return c
    sys.exit("google-chrome/chromium bulunamadı")


def uret(ad, fn, t, sonek):
    hedef = DIZIN / f"market-{ad}{sonek}.png"
    with tempfile.TemporaryDirectory() as tmp:
        htm = Path(tmp) / "s.html"
        htm.write_text(sayfa(fn(t)), encoding="utf-8")
        ham = Path(tmp) / "ham.png"
        # 2x render + küçültme: 1x'te metin kenarları tırtıklı çıkıyor.
        subprocess.run([chrome_bul(), "--headless=new", "--disable-gpu", "--no-sandbox",
                        "--hide-scrollbars", "--force-device-scale-factor=2",
                        f"--window-size={GEN[0]},{GEN[1]}", f"--screenshot={ham}",
                        f"--user-data-dir={tmp}/prof", htm.as_uri()],
                       check=True, capture_output=True)
        subprocess.run(["convert", str(ham), "-resize", f"{GEN[0]}x{GEN[1]}",
                        "-strip", "-quality", "95", str(hedef)], check=True)
    print(f"  {hedef.name}")


if __name__ == "__main__":
    print("1200x760 Marketplace görselleri:")
    for ad, fn in GORSELLER.items():
        uret(ad, fn, EN, "")
        uret(ad, fn, TR, ".tr")
