#!/usr/bin/env python3
"""updatePlugins.xml üretir (custom plugin repository beslemesi).

    python3 updateplugins_uret.py <surum> > build/distributions/updatePlugins.xml

Açıklama plugin.xml'deki CDATA bloğundan OKUNUR, burada elle yazılmaz: elle
yazılan tek satırlık özet IDE'nin eklenti ekranında tek satır olarak görünüyordu
(2026-09-10, kullanıcı ekran görüntüsü) — Marketplace'teki 3210 karakterlik
açıklamanın yanında kayıt bomboş duruyordu.
"""
import html
import re
import sys
from pathlib import Path

DIZIN = Path(__file__).resolve().parent
INDIRME = "https://github.com/muslu/cem/releases/latest/download/cem-intellij.zip"


def plugin_xml_alanlari():
    xml = (DIZIN / "src/main/resources/META-INF/plugin.xml").read_text(encoding="utf-8")
    desc = re.search(r"<description>\s*<!\[CDATA\[(.*?)\]\]>\s*</description>", xml, re.S)
    if not desc:
        sys.exit("plugin.xml içinde CDATA'lı <description> bulunamadı")
    pid = re.search(r"<id>([^<]+)</id>", xml).group(1)
    ad = re.search(r"<name>([^<]+)</name>", xml).group(1)
    satici = re.search(r"<vendor[^>]*>([^<]+)</vendor>", xml).group(1)
    since = re.search(r'sinceBuild\s*=\s*"([^"]+)"', (DIZIN / "build.gradle.kts").read_text(encoding="utf-8"))
    return pid, ad, satici, desc.group(1).strip(), (since.group(1) if since else "233")


def main():
    if len(sys.argv) != 2:
        sys.exit(f"kullanım: {Path(sys.argv[0]).name} <surum>")
    surum = sys.argv[1]
    pid, ad, satici, desc, since = plugin_xml_alanlari()
    print(f"""<?xml version="1.0" encoding="UTF-8"?>
<plugins>
  <plugin id="{html.escape(pid)}"
          url="{INDIRME}"
          version="{html.escape(surum)}">
    <idea-version since-build="{html.escape(since)}"/>
    <name>{html.escape(ad)}</name>
    <vendor>{html.escape(satici)}</vendor>
    <description><![CDATA[
{desc}
    ]]></description>
  </plugin>
</plugins>""")


if __name__ == "__main__":
    main()
