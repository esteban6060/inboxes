import { describe, expect, it } from "vitest";
import { Editor } from "@tiptap/core";
import StarterKit from "@tiptap/starter-kit";
import { InlineImage, IMAGE_WIDTHS } from "../inline-image-extension";

describe("inline email image", () => {
  it("keeps its place between paragraphs and persists size and alignment", () => {
    const editor = new Editor({
      extensions: [StarterKit, InlineImage],
      content: "<p>Before</p><p>After</p>",
    });
    const position = editor.state.doc.child(0).nodeSize;
    editor.commands.insertContentAt(position, {
      type: "inlineImage",
      attrs: {
        src: "/api/attachments/11111111-1111-4111-8111-111111111111/inline",
        alt: "photo.png",
        width: IMAGE_WIDTHS.L,
        align: "center",
      },
    });

    const html = editor.getHTML();
    expect(html.indexOf("Before")).toBeLessThan(html.indexOf("<img"));
    expect(html.indexOf("<img")).toBeLessThan(html.indexOf("After"));
    expect(html).toContain('width="600"');
    expect(html).toContain("text-align: center");

    const restored = new Editor({ extensions: [StarterKit, InlineImage], content: html });
    expect(restored.state.doc.child(1).attrs.width).toBe(600);
    expect(restored.state.doc.child(1).attrs.align).toBe("center");
    restored.destroy();
    editor.destroy();
  });
});
