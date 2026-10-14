import { Node, mergeAttributes } from "@tiptap/core";

export const IMAGE_WIDTHS = { S: 160, M: 320, L: 600 } as const;
export type ImageSize = keyof typeof IMAGE_WIDTHS;
export type ImageAlign = "start" | "center" | "end";

const validWidth = (value: string | null) =>
  Object.values(IMAGE_WIDTHS).includes(Number(value) as 160 | 320 | 600)
    ? Number(value)
    : IMAGE_WIDTHS.M;

export const InlineImage = Node.create({
  name: "inlineImage",
  group: "block",
  atom: true,
  selectable: true,
  addAttributes() {
    return {
      src: { default: "", parseHTML: (element) => element.querySelector("img")?.getAttribute("src") || "" },
      alt: { default: "Image", parseHTML: (element) => element.querySelector("img")?.getAttribute("alt") || "Image" },
      width: { default: IMAGE_WIDTHS.M, parseHTML: (element) => validWidth(element.querySelector("img")?.getAttribute("width") || null) },
      align: {
        default: "start",
        parseHTML: (element) => {
          const align = (element as HTMLElement).style.textAlign;
          return align === "center" ? "center" : align === "end" || align === "right" ? "end" : "start";
        },
      },
    };
  },
  parseHTML() {
    return [{ tag: "div[data-inline-image]" }];
  },
  renderHTML({ HTMLAttributes }) {
    const { src, alt, width, align } = HTMLAttributes;
    const safeWidth = validWidth(String(width));
    const safeAlign = align === "center" ? "center" : align === "end" ? "right" : "left";
    return [
      "div",
      { "data-inline-image": "", style: `text-align: ${safeAlign}; margin: 8px 0` },
      ["img", mergeAttributes({ src, alt, width: safeWidth, style: `display: inline-block; width: ${safeWidth}px; max-width: 100%; height: auto` })],
    ];
  },
});
