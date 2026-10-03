import { describe, expect, it } from "vitest";
import { getPhotoCrop } from "../photo-crop";

describe("photo crop geometry", () => {
  it("centers a landscape photo without stretching", () => {
    expect(getPhotoCrop(1200, 800, 1, 0, 0)).toEqual({ x: 200, y: 0, size: 800 });
  });
  it("zooms around the center of a portrait", () => {
    expect(getPhotoCrop(800, 1200, 2, 0, 0)).toEqual({ x: 200, y: 400, size: 400 });
  });
  it("moves to either edge without leaving the image", () => {
    expect(getPhotoCrop(1200, 800, 2, 1, 1)).toEqual({ x: 0, y: 0, size: 400 });
    expect(getPhotoCrop(1200, 800, 2, -1, -1)).toEqual({ x: 800, y: 400, size: 400 });
  });
  it("bounds the crop for out-of-range controls", () => {
    expect(getPhotoCrop(1200, 800, 0.5, 8, -8)).toEqual({ x: 0, y: 0, size: 800 });
  });
});
