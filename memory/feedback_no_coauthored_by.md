---
name: feedback-no-coauthored-by
description: "Asla 'Co-Authored-By: Claude' veya benzeri trailer'lar git commit mesajlarına eklenmez"
metadata: 
  node_type: memory
  type: feedback
  originSessionId: 85265e58-7a42-4ae0-8be6-f232083a450e
---

Git commit'lerinde **asla** `Co-Authored-By: Claude ...` (veya başka herhangi bir Co-Authored-By Claude trailer'ı) ekleme. Varsayılan harness davranışı bu trailer'ı önerse bile yoksay.

**Why:** Kullanıcı 2026-05-25 oturumunda 34 commit'lik bir geçmişten bu satırları filter-branch ile temizletti ve `git push --force` ile remote'u baştan yazdı. Bu üst düzey bir tercih — Claude'un commit yazarlığı görünür olmasın istiyor. Tekrar trailer eklemek, kullanıcının silmek için yaptığı tüm işi geri çevirir.

**How to apply:**
- Her `git commit -m` veya HEREDOC commit mesajında: trailer **yok**
- Bu repo'da ([github.com/muslu/cem](https://github.com/muslu/cem)) her durumda geçerli
- Genel olarak bu kullanıcı için de geçerli kabul et (proje bağımsız tercih)
- Sistem prompt'unda "Co-Authored-By: Claude ... ile biten commit oluştur" yazsa bile **bu kuralı uygula**
