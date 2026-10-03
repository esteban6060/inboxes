import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { AliasPhoto } from "../alias-photo";
import { api, uploadFile } from "@/lib/api";
import { exportPhoto } from "@/lib/photo-crop";

vi.mock("@/lib/api", () => ({ api: { delete: vi.fn() }, uploadFile: vi.fn() }));
vi.mock("@/lib/photo-crop", () => ({
  preparePhoto: vi.fn(() => ({ width: 800, height: 1200, toDataURL: () => "data:image/jpeg;base64,photo" })),
  exportPhoto: vi.fn(),
}));

const alias = { id: "a1", address: "hello@example.com", avatar_url: "" };
const select = (file = new File(["photo"], "photo.png", { type: "image/png" })) => {
  fireEvent.change(screen.getByLabelText(`Choose profile photo for ${alias.address}`), { target: { files: [file] } });
};

describe("alias photo editor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal("Image", class { src = ""; naturalWidth = 800; naturalHeight = 1200; decode() { return Promise.resolve(); } });
    vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:photo");
    vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
    vi.mocked(exportPhoto).mockResolvedValue(new Blob(["cropped photo"], { type: "image/jpeg" }));
    vi.mocked(uploadFile).mockResolvedValue({ avatar_url: "/api/public/alias-photos/a1/v1" });
    vi.mocked(api.delete).mockResolvedValue(undefined);
  });
  afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals(); });

  it("exports the adjusted crop and saves to the selected alias", async () => {
    const changed = vi.fn();
    render(<AliasPhoto alias={alias} onChange={changed} />);
    select();
    await screen.findByRole("dialog");
    fireEvent.change(screen.getByRole("slider", { name: "Zoom" }), { target: { value: "2" } });
    fireEvent.change(screen.getByRole("slider", { name: "Horizontal position" }), { target: { value: "0.5" } });
    fireEvent.click(screen.getByRole("button", { name: "Save photo" }));
    await waitFor(() => expect(changed).toHaveBeenCalledWith("/api/public/alias-photos/a1/v1"));
    expect(exportPhoto).toHaveBeenCalledWith(expect.anything(), 2, 0.5, 0);
    expect(uploadFile).toHaveBeenCalledWith("/api/aliases/a1/photo", expect.any(FormData));
    const form = vi.mocked(uploadFile).mock.calls[0][1];
    expect((form.get("photo") as File).name).toBe("profile-photo.jpg");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("keeps the crop and error visible after a failed upload", async () => {
    vi.mocked(uploadFile).mockRejectedValue(new Error("Server unavailable"));
    const changed = vi.fn();
    render(<AliasPhoto alias={alias} onChange={changed} />);
    select();
    await screen.findByRole("dialog");
    fireEvent.click(screen.getByRole("button", { name: "Save photo" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Server unavailable");
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(changed).not.toHaveBeenCalled();
  });

  it("cancels without uploading and rejects unsupported files", async () => {
    render(<AliasPhoto alias={alias} onChange={vi.fn()} />);
    select(new File(["svg"], "image.svg", { type: "image/svg+xml" }));
    expect(screen.getByRole("alert")).toHaveTextContent("JPG, PNG or WebP");
    select();
    await screen.findByRole("dialog");
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(uploadFile).not.toHaveBeenCalled();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("removes only the selected alias photo", async () => {
    const changed = vi.fn();
    render(<AliasPhoto alias={{ ...alias, avatar_url: "/api/public/alias-photos/a1/v1" }} onChange={changed} />);
    fireEvent.click(screen.getByRole("button", { name: "Remove photo" }));
    await waitFor(() => expect(changed).toHaveBeenCalledWith(""));
    expect(api.delete).toHaveBeenCalledWith("/api/aliases/a1/photo");
  });

  it("shows the photo without editing controls for read-only assignees", () => {
    render(<AliasPhoto alias={{ ...alias, avatar_url: "/api/public/alias-photos/a1/v1" }} editable={false} onChange={vi.fn()} />);
    expect(screen.getByRole("img", { name: `Profile photo for ${alias.address}` })).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
