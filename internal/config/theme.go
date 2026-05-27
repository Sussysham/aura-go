package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Theme struct {
	Name      string
	Header    string
	Accent    string
	Selection string
	Success   string
	SuccessSt string
	Warning   string
	WarningSt string
	Error     string
	ErrorSt   string
	Muted     string
	Violet    string
	VioletSt  string
	SelectRed string
	SelectGrn string
}

var ActiveTheme *Theme

func HexToTrueColor(hex string) string {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 6 {
		r, _ := strconv.ParseUint(hex[0:2], 16, 8)
		g, _ := strconv.ParseUint(hex[2:4], 16, 8)
		b, _ := strconv.ParseUint(hex[4:6], 16, 8)
		return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
	}
	return ""
}

func HexToTrueColorBg(hex string) string {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 6 {
		r, _ := strconv.ParseUint(hex[0:2], 16, 8)
		g, _ := strconv.ParseUint(hex[2:4], 16, 8)
		b, _ := strconv.ParseUint(hex[4:6], 16, 8)
		return fmt.Sprintf("\033[48;2;%d;%d;%dm", r, g, b)
	}
	return ""
}

func SupportsTrueColor() bool {
	colorterm := os.Getenv("COLORTERM")
	return colorterm == "truecolor" || colorterm == "24bit"
}

func GetTheme(name string, custom CustomThemeConfig) *Theme {
	useTrueColor := SupportsTrueColor()

	t := &Theme{Name: name}

	switch strings.ToLower(name) {
	case "nord":
		if useTrueColor {
			t.Header = HexToTrueColor("#81A1C1")
			t.Accent = HexToTrueColor("#88C0D0")
			t.Selection = HexToTrueColorBg("#4C566A") + "\033[38;2;229;233;240m"
			t.Success = HexToTrueColor("#A3BE8C")
			t.SuccessSt = t.Success
			t.Warning = HexToTrueColor("#EBCB8B")
			t.WarningSt = t.Warning
			t.Error = HexToTrueColor("#BF616A")
			t.ErrorSt = t.Error
			t.Muted = HexToTrueColor("#4C566A")
			t.Violet = HexToTrueColor("#B48EAD")
			t.VioletSt = t.Violet
			t.SelectRed = HexToTrueColorBg("#BF616A") + "\033[38;2;229;233;240m"
			t.SelectGrn = HexToTrueColorBg("#A3BE8C") + "\033[38;2;46;52;64m"
		} else {
			t.Header = "\033[1;34m"
			t.Accent = "\033[36m"
			t.Selection = "\033[7;36m"
			t.Success = "\033[1;32m"
			t.SuccessSt = "\033[32m"
			t.Warning = "\033[1;33m"
			t.WarningSt = "\033[33m"
			t.Error = "\033[1;31m"
			t.ErrorSt = "\033[31m"
			t.Muted = "\033[38;5;244m"
			t.Violet = "\033[1;35m"
			t.VioletSt = "\033[35m"
			t.SelectRed = "\033[7;31m"
			t.SelectGrn = "\033[7;32m"
		}
	case "dracula":
		if useTrueColor {
			t.Header = HexToTrueColor("#BD93F9")
			t.Accent = HexToTrueColor("#FF79C6")
			t.Selection = HexToTrueColorBg("#44475A") + "\033[38;2;248;248;242m"
			t.Success = HexToTrueColor("#50FA7B")
			t.SuccessSt = t.Success
			t.Warning = HexToTrueColor("#F1FA8C")
			t.WarningSt = t.Warning
			t.Error = HexToTrueColor("#FF5555")
			t.ErrorSt = t.Error
			t.Muted = HexToTrueColor("#6272A4")
			t.Violet = HexToTrueColor("#BD93F9")
			t.VioletSt = t.Violet
			t.SelectRed = HexToTrueColorBg("#FF5555") + "\033[38;2;248;248;242m"
			t.SelectGrn = HexToTrueColorBg("#50FA7B") + "\033[38;2;40;42;54m"
		} else {
			t.Header = "\033[1;35m"
			t.Accent = "\033[35m"
			t.Selection = "\033[7;35m"
			t.Success = "\033[1;32m"
			t.SuccessSt = "\033[32m"
			t.Warning = "\033[1;33m"
			t.WarningSt = "\033[33m"
			t.Error = "\033[1;31m"
			t.ErrorSt = "\033[31m"
			t.Muted = "\033[38;5;244m"
			t.Violet = "\033[1;35m"
			t.VioletSt = "\033[35m"
			t.SelectRed = "\033[7;31m"
			t.SelectGrn = "\033[7;32m"
		}
	case "solarized":
		if useTrueColor {
			t.Header = HexToTrueColor("#268BD2")
			t.Accent = HexToTrueColor("#2AA198")
			t.Selection = HexToTrueColorBg("#073642") + "\033[38;2;147;161;161m"
			t.Success = HexToTrueColor("#859900")
			t.SuccessSt = t.Success
			t.Warning = HexToTrueColor("#B58900")
			t.WarningSt = t.Warning
			t.Error = HexToTrueColor("#DC322F")
			t.ErrorSt = t.Error
			t.Muted = HexToTrueColor("#586E75")
			t.Violet = HexToTrueColor("#D33682")
			t.VioletSt = t.Violet
			t.SelectRed = HexToTrueColorBg("#DC322F") + "\033[38;2;253;246;227m"
			t.SelectGrn = HexToTrueColorBg("#859900") + "\033[38;2;7;54;66m"
		} else {
			t.Header = "\033[1;34m"
			t.Accent = "\033[36m"
			t.Selection = "\033[7;36m"
			t.Success = "\033[1;32m"
			t.SuccessSt = "\033[32m"
			t.Warning = "\033[1;33m"
			t.WarningSt = "\033[33m"
			t.Error = "\033[1;31m"
			t.ErrorSt = "\033[31m"
			t.Muted = "\033[38;5;244m"
			t.Violet = "\033[1;35m"
			t.VioletSt = "\033[35m"
			t.SelectRed = "\033[7;31m"
			t.SelectGrn = "\033[7;32m"
		}
	case "gruvbox":
		if useTrueColor {
			t.Header = HexToTrueColor("#FE8019")
			t.Accent = HexToTrueColor("#8EC07C")
			t.Selection = HexToTrueColorBg("#3C3836") + "\033[38;2;235;219;178m"
			t.Success = HexToTrueColor("#B8BB26")
			t.SuccessSt = t.Success
			t.Warning = HexToTrueColor("#FABD2F")
			t.WarningSt = t.Warning
			t.Error = HexToTrueColor("#FB4934")
			t.ErrorSt = t.Error
			t.Muted = HexToTrueColor("#928374")
			t.Violet = HexToTrueColor("#D3869B")
			t.VioletSt = t.Violet
			t.SelectRed = HexToTrueColorBg("#FB4934") + "\033[38;2;235;219;178m"
			t.SelectGrn = HexToTrueColorBg("#B8BB26") + "\033[38;2;28;28;28m"
		} else {
			t.Header = "\033[1;33m"
			t.Accent = "\033[36m"
			t.Selection = "\033[7;33m"
			t.Success = "\033[1;32m"
			t.SuccessSt = "\033[32m"
			t.Warning = "\033[1;33m"
			t.WarningSt = "\033[33m"
			t.Error = "\033[1;31m"
			t.ErrorSt = "\033[31m"
			t.Muted = "\033[38;5;244m"
			t.Violet = "\033[1;35m"
			t.VioletSt = "\033[35m"
			t.SelectRed = "\033[7;31m"
			t.SelectGrn = "\033[7;32m"
		}
	case "contrast":
		t.Header = "\033[1;37m"
		t.Accent = "\033[37m"
		t.Selection = "\033[7;37m"
		t.Success = "\033[1;37m"
		t.SuccessSt = "\033[37m"
		t.Warning = "\033[1;37m"
		t.WarningSt = "\033[37m"
		t.Error = "\033[1;37m"
		t.ErrorSt = "\033[37m"
		t.Muted = "\033[38;5;244m"
		t.Violet = "\033[1;37m"
		t.VioletSt = "\033[37m"
		t.SelectRed = "\033[7;37m"
		t.SelectGrn = "\033[7;37m"
	case "custom":
		if useTrueColor && custom.Header != "" {
			t.Header = HexToTrueColor(custom.Header)
			t.Accent = HexToTrueColor(custom.Accent)
			t.Selection = HexToTrueColorBg(custom.Selection) + "\033[38;2;255;255;255m"
			t.Success = HexToTrueColor(custom.Success)
			t.SuccessSt = t.Success
			t.Warning = HexToTrueColor(custom.Warning)
			t.WarningSt = t.Warning
			t.Error = HexToTrueColor(custom.Error)
			t.ErrorSt = t.Error
			t.Muted = HexToTrueColor(custom.Muted)
			t.Violet = HexToTrueColor(custom.Violet)
			t.VioletSt = t.Violet
			t.SelectRed = HexToTrueColorBg(custom.Error) + "\033[38;2;255;255;255m"
			t.SelectGrn = HexToTrueColorBg(custom.Success) + "\033[38;2;0;0;0m"
		} else {
			// Fallback to Midnight
			t.Header = "\033[1;36m"
			t.Accent = "\033[36m"
			t.Selection = "\033[7;36m"
			t.Success = "\033[1;32m"
			t.SuccessSt = "\033[32m"
			t.Warning = "\033[1;33m"
			t.WarningSt = "\033[33m"
			t.Error = "\033[1;31m"
			t.ErrorSt = "\033[31m"
			t.Muted = "\033[38;5;244m"
			t.Violet = "\033[1;35m"
			t.VioletSt = "\033[35m"
			t.SelectRed = "\033[7;31m"
			t.SelectGrn = "\033[7;32m"
		}
	default: // Midnight (classic cyan/violet)
		if useTrueColor {
			t.Header = HexToTrueColor("#7B2FBE") // Violet
			t.Accent = HexToTrueColor("#00D4FF") // Cyan
			t.Selection = HexToTrueColorBg("#7B2FBE") + "\033[38;2;255;255;255m"
			t.Success = HexToTrueColor("#00E676") // Green
			t.SuccessSt = t.Success
			t.Warning = HexToTrueColor("#FFD600") // Yellow
			t.WarningSt = t.Warning
			t.Error = HexToTrueColor("#FF1744") // Red
			t.ErrorSt = t.Error
			t.Muted = HexToTrueColor("#666666")
			t.Violet = HexToTrueColor("#9C27B0")
			t.VioletSt = t.Violet
			t.SelectRed = HexToTrueColorBg("#FF1744") + "\033[38;2;255;255;255m"
			t.SelectGrn = HexToTrueColorBg("#00E676") + "\033[38;2;0;0;0m"
		} else {
			t.Header = "\033[1;36m"
			t.Accent = "\033[36m"
			t.Selection = "\033[7;36m"
			t.Success = "\033[1;32m"
			t.SuccessSt = "\033[32m"
			t.Warning = "\033[1;33m"
			t.WarningSt = "\033[33m"
			t.Error = "\033[1;31m"
			t.ErrorSt = "\033[31m"
			t.Muted = "\033[38;5;244m"
			t.Violet = "\033[1;35m"
			t.VioletSt = "\033[35m"
			t.SelectRed = "\033[7;31m"
			t.SelectGrn = "\033[7;32m"
		}
	}
	return t
}

func LoadTheme(name string, custom CustomThemeConfig) {
	ActiveTheme = GetTheme(name, custom)
}

func init() {
	LoadTheme("midnight", CustomThemeConfig{})
}
