"""
gaea 内嵌的 Excel 公式重算脚本（xlsxedit.Recalc 经 python 调用）。

来历：吸收自 xlsx 技能的 recalc.py。原先进在 ~/.codex/skills 或工作区 .gaea/skills
——脚本不是 gaea 的：各工作区各一份、改仓库那份修不到别的工作区。2026-11-26 起
随 gaea 二进制内嵌（internal/office/xlsxedit/recalc.go），自包含，不再 import
技能包的 office.soffice。相对技能包那份的三处改动：

1. 独立配置目录。Go 层用环境变量 GAEA_RECALC_PROFILE 给一个目录，就用
   -env:UserInstallation 指过去——每次调用一个自己的 LibreOffice 实例。走默认
   配置目录时，只要机器上另有一个 LibreOffice 实例活着，本次调用就被转交给它并
   立刻返回，脚本随即读到没重算的文件却回「成功」（2026-11-26 实测，6/6 复现）；
   独立目录后同一场景 3/3 正确重算。
2. 失败回执带 stage：LibreOffice 环节 = recalc，重算后回读工作簿异常 = read
   （zip 解压类瞬时错误）。Go 层据此自动重试，不再把人引去查单元格。
3. 去掉 Linux 沙箱的 LD_PRELOAD 垫片（gaea 桌面端用不到，运行时编 C 风险大）。

用法：python recalc.py <excel 文件> [超时秒数]
输出：一行 JSON 对象——成功 {"status", "total_errors", "error_summary",
"total_formulas"}；失败 {"error": "...", "stage": "recalc" | "read"}。
"""

import json
import os
import platform
import subprocess
import sys
from pathlib import Path

from openpyxl import load_workbook

MACRO_DIR_MACOS = "~/Library/Application Support/LibreOffice/4/user/basic/Standard"
MACRO_DIR_LINUX = "~/.config/libreoffice/4/user/basic/Standard"
MACRO_DIR_WINDOWS = os.path.join(os.environ.get("APPDATA", ""), "LibreOffice", "4", "user", "basic", "Standard")
MACRO_FILENAME = "Module1.xba"
MACRO_URL = "vnd.sun.star.script:Standard.Module1.RecalculateAndSave?language=Basic&location=application"

# Go 层放进来的本次调用专属配置目录；没有这个变量就走 LibreOffice 默认配置目录。
PROFILE_ENV = "GAEA_RECALC_PROFILE"

RECALCULATE_MACRO = """<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE script:module PUBLIC "-//OpenOffice.org//DTD OfficeDocument 1.0//EN" "module.dtd">
<script:module xmlns:script="http://openoffice.org/2000/script" script:name="Module1" script:language="StarBasic">
    Sub RecalculateAndSave()
      ThisComponent.calculateAll()
      ThisComponent.store()
      ThisComponent.close(True)
    End Sub
</script:module>"""


def find_soffice():
    """定位 LibreOffice 可执行文件：常见安装位置优先，再退 PATH。"""
    if os.name == "nt":
        candidates = [
            os.path.join(os.environ.get("ProgramFiles", "C:/Program Files"), "LibreOffice", "program", "soffice.exe"),
            os.path.join(os.environ.get("ProgramFiles(x86)", "C:/Program Files (x86)"), "LibreOffice", "program", "soffice.exe"),
            os.path.join(os.environ.get("LOCALAPPDATA", ""), "Programs", "LibreOffice", "program", "soffice.exe"),
        ]
        for c in candidates:
            if c and os.path.isfile(c):
                return c
    return "soffice"


def soffice_env():
    env = os.environ.copy()
    env["SAL_USE_VCLPLUGIN"] = "svp"
    return env


def isolated_profile():
    """返回 (soffice 的配置目录参数, 宏所在目录)。没给 GAEA_RECALC_PROFILE 就是默认
    配置目录，参数表为空。"""
    prof = os.environ.get(PROFILE_ENV, "").strip()
    if prof:
        root = Path(prof).resolve()
        return ["-env:UserInstallation=" + root.as_uri()], str(root / "user" / "basic" / "Standard")
    system = platform.system()
    if system == "Darwin":
        macro_dir = os.path.expanduser(MACRO_DIR_MACOS)
    elif system == "Windows":
        macro_dir = MACRO_DIR_WINDOWS
    else:
        macro_dir = os.path.expanduser(MACRO_DIR_LINUX)
    return [], macro_dir


def has_gtimeout():
    try:
        subprocess.run(
            ["gtimeout", "--version"], capture_output=True, timeout=1, check=False
        )
        return True
    except (FileNotFoundError, subprocess.TimeoutExpired):
        return False


def setup_libreoffice_macro(macro_dir, env_args):
    macro_file = os.path.join(macro_dir, MACRO_FILENAME)

    if os.path.exists(macro_file) and "RecalculateAndSave" in Path(macro_file).read_text(
        encoding="utf-8", errors="replace"
    ):
        return True

    if not os.path.exists(macro_dir):
        # 让 LibreOffice 自己把配置目录初始化出来再往里放宏。手写一个最小目录
        # 不行：LibreOffice 对没经它初始化的配置目录会挂住不返回（2026-11-26 实测）。
        try:
            subprocess.run(
                [find_soffice(), *env_args, "--headless", "--terminate_after_init"],
                capture_output=True,
                timeout=30,
                env=soffice_env(),
            )
        except (OSError, subprocess.TimeoutExpired):
            return False
        os.makedirs(macro_dir, exist_ok=True)

    try:
        Path(macro_file).write_text(RECALCULATE_MACRO, encoding="utf-8")
        return True
    except OSError:
        return False


def recalc(filename, timeout=30):
    if not Path(filename).exists():
        return {"error": "File " + filename + " does not exist", "stage": "recalc"}

    abs_path = str(Path(filename).absolute())
    env_args, macro_dir = isolated_profile()
    if not setup_libreoffice_macro(macro_dir, env_args):
        return {"error": "Failed to setup LibreOffice macro", "stage": "recalc"}

    cmd = [find_soffice(), *env_args, "--headless", "--norestore", MACRO_URL, abs_path]

    system = platform.system()
    if system == "Linux":
        cmd = ["timeout", str(timeout)] + cmd
    elif system == "Darwin" and has_gtimeout():
        cmd = ["gtimeout", str(timeout)] + cmd

    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            errors="replace",
            env=soffice_env(),
            timeout=timeout if system == "Windows" else None,
        )
    except subprocess.TimeoutExpired:
        return {"error": "LibreOffice recalculation timed out after %ds" % timeout, "stage": "recalc"}
    except OSError as exc:
        return {"error": "LibreOffice could not be started: %s" % exc, "stage": "recalc"}

    if result.returncode != 0 and result.returncode != 124:
        detail = (result.stderr or "").strip() or "Unknown error during recalculation"
        if "Module1" in detail or "RecalculateAndSave" in detail:
            detail = "LibreOffice macro not configured properly: " + detail
        return {"error": "LibreOffice exited with code %d: %s" % (result.returncode, detail[:400]), "stage": "recalc"}

    try:
        wb = load_workbook(filename, data_only=True)

        excel_errors = [
            "#VALUE!",
            "#DIV/0!",
            "#REF!",
            "#NAME?",
            "#NULL!",
            "#NUM!",
            "#N/A",
        ]
        error_details = {err: [] for err in excel_errors}
        total_errors = 0

        for sheet_name in wb.sheetnames:
            ws = wb[sheet_name]
            for row in ws.iter_rows():
                for cell in row:
                    if cell.value is not None and isinstance(cell.value, str):
                        for err in excel_errors:
                            if err in cell.value:
                                location = "%s!%s" % (sheet_name, cell.coordinate)
                                error_details[err].append(location)
                                total_errors += 1
                                break

        wb.close()

        result = {
            "status": "success" if total_errors == 0 else "errors_found",
            "total_errors": total_errors,
            "error_summary": {},
        }

        for err_type, locations in error_details.items():
            if locations:
                result["error_summary"][err_type] = {
                    "count": len(locations),
                    "locations": locations[:20],
                }

        wb_formulas = load_workbook(filename, data_only=False)
        formula_count = 0
        for sheet_name in wb_formulas.sheetnames:
            ws = wb_formulas[sheet_name]
            for row in ws.iter_rows():
                for cell in row:
                    if (
                        cell.value
                        and isinstance(cell.value, str)
                        and cell.value.startswith("=")
                    ):
                        formula_count += 1
        wb_formulas.close()

        result["total_formulas"] = formula_count

        return result

    except Exception as e:
        return {"error": str(e), "stage": "read"}


def main():
    if len(sys.argv) < 2:
        print("Usage: python recalc.py <excel_file> [timeout_seconds]")
        print("Recalculates all formulas in an Excel file using LibreOffice; prints one JSON object.")
        sys.exit(1)

    filename = sys.argv[1]
    timeout = int(sys.argv[2]) if len(sys.argv) > 2 else 30

    print(json.dumps(recalc(filename, timeout), indent=2))


if __name__ == "__main__":
    main()
