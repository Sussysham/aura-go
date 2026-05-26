# 📖 Aura TUI

A statically compiled, hyper-secure, high-performance, and **completely local Terminal User Interface (TUI)** Ebook and Document Library Reader. 

Aura is designed specifically as a cognitive and behavioral intervention to defeat "digital hoarding" (collecting books without engaging with them). It leverages active Work-In-Progress (WIP) focus decks, stale inbox surfacing, real-time keyboard navigation, and an inline, console-native ebook reader.

---

## ✨ Features

*   **Zero-Config Portability**: Run it anywhere out of the box. Aura dynamically detects local catalogs and creates isolated state databases instantly in the current directory.
*   **Fully Responsive Layout**: Divider lines, progress bars, card margins, and scrollable table viewports reflow in real time to fit small, maximized, or split terminal windows.
*   **Auto-Discovery Directory Scanner**: Press `i` to recursively scan any directory for `.epub`, `.pdf`, and `.txt` files.
*   **Quick content-hash fingerprinting**: Identifies duplicates by hashing the first 32KB of content in less than 0.1ms per file, instantly grouping duplicates for visual side-by-side comparison and pruning.
*   **Dynamic Ebook Re-wrapping**: Ebook text is dynamically re-wrapped on the fly inside the Console Ebook Reader when terminal widths change, optimizing readable column spans.
*   **Strict WIP caps**: Caps your active "Reading Now" slots to a maximum of 2, encouraging focused, dedicated deep engagement.
*   **SumatraPDF Integration & Fallbacks**: Natively queries Windows directories to launch SumatraPDF, with graceful fallbacks to the OS-default PDF viewer if SumatraPDF is not installed.
*   **PolyForm License Protection**: Released under the legally ironclad **PolyForm Noncommercial 1.0.0** license—completely free for personal use but strictly protected against commercial exploitation.

---

## ⚡ Zero-Friction Installation (Windows)

For standard users, you can install Aura TUI globally with a single, one-liner command in standard PowerShell:

```powershell
iex (iwr -UseBasicParsing https://raw.githubusercontent.com/Sussysham/aura-go/main/install.ps1)
```

This installer automatically detects your processor architecture (64-bit standard, ARM64, or legacy 32-bit), downloads the correct pre-compiled binary, registers it inside your local user AppData folder, and appends it to your User environment `%PATH%` so you can run `aura` from any terminal globally!

---

## 🛠️ Building & Cross-Compilation (Developers)

If you have Go 1.26+ installed, you can compile natively from the source repository:

```powershell
# Compile local architecture binary
go build -o aura.exe main.go

# Compile for all Windows architectures at once (dist/)
.\build_releases.ps1
```

---

## 🕹️ Keyboard Shortcuts

Aura is completely keyboard-driven for maximum speed:

| Key(s) | Action | Target View |
| :--- | :--- | :--- |
| **`Up / Down` / `j / k`** | Scroll selection highlight up and down through lists | All Lists |
| **`Enter`** | Launch the selected book in SumatraPDF or OS default viewer | Explorer |
| **`v`** | Natively view EPUB/TXT files inside your Console Reader | Explorer |
| **`w`** | Cycle book status: `Inbox` ➔ `Reading` ➔ `Read` ➔ `Reference` | Explorer |
| **`s`** | **Search Filter**: Type keywords, `Esc` to clear, `Enter` to lock | Explorer |
| **`i`** | **Ingest Scanner**: Crawl directory recursively for ebooks | Explorer |
| **`d`** | **Stats Dashboard**: View dynamic behavioral charts & metrics | Explorer |
| **`u`** | **Duplicates Deck**: Group, compare, and prune duplicates | Explorer |
| **`Esc`** | Cancel actions, clear inputs, or return to Explorer | All Views |
| **`q`** | Safely restore console mode and exit Aura TUI | All Views |

---

## 📬 Contact

For questions, feedback, or collaboration inquiries, reach out at **[schighproton.me](https://schighproton.me)**.

---

## ⚖️ License

Aura TUI is released under the **PolyForm Noncommercial License 1.0.0**. You are free to copy, modify, distribute, and run Aura TUI globally for personal, educational, research, and non-commercial purposes. **All commercial exploitation, commercial hosting (SaaS), or commercial development is strictly prohibited.**

See the [LICENSE](LICENSE) file for the full legal text.
