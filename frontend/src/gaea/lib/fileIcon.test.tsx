import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { FileTypeIcon } from "./fileIcon";

function kind(name: string, isDir = false, open = false): string | null {
  const { container, unmount } = render(<FileTypeIcon name={name} isDir={isDir} open={open} />);
  const value = container.querySelector("[data-file-kind]")?.getAttribute("data-file-kind") ?? null;
  unmount();
  return value;
}

describe("FileTypeIcon", () => {
  it("maps office, code, config, archive, image and database files", () => {
    expect(kind("报告.docx")).toBe("word");
    expect(kind("预算.xlsx")).toBe("sheet");
    expect(kind("汇报.pptx")).toBe("slides");
    expect(kind("规格.pdf")).toBe("pdf");
    expect(kind("main.go")).toBe("code");
    expect(kind("settings.yaml")).toBe("config");
    expect(kind("assets.zip")).toBe("archive");
    expect(kind("photo.webp")).toBe("image");
    expect(kind("store.sqlite")).toBe("database");
  });

  it("distinguishes open folders and text/binary fallbacks", () => {
    expect(kind("src", true, false)).toBe("folder");
    expect(kind("src", true, true)).toBe("folder-open");
    expect(kind("README.md")).toBe("text");
    expect(kind("payload.bin")).toBe("file");
  });
});
