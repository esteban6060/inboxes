"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Camera, RotateCw, UserRound } from "lucide-react";
import { api, uploadFile } from "@/lib/api";
import { exportPhoto, preparePhoto } from "@/lib/photo-crop";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { Spinner } from "@/components/ui/spinner";
import { cn } from "@/lib/utils";

export function AliasAvatar({ url, address, className }: { url?: string; address: string; className?: string }) {
  const [failed, setFailed] = useState(false);
  useEffect(() => setFailed(false), [url]);
  return (
    <span className={cn("flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-full bg-muted", className)}>
      {url && !failed ? (
        <img src={`${process.env.NEXT_PUBLIC_API_URL || ""}${url}`} alt={`Profile photo for ${address}`} className="h-full w-full object-cover" onError={() => setFailed(true)} />
      ) : <UserRound className="h-1/2 w-1/2 text-muted-foreground" aria-hidden="true" />}
    </span>
  );
}

interface AliasPhotoProps {
  alias: { id: string; address: string; avatar_url?: string };
  editable?: boolean;
  onChange: (url: string) => void;
}

export function AliasPhoto({ alias, editable = true, onChange }: AliasPhotoProps) {
  const input = useRef<HTMLInputElement>(null);
  const original = useRef<HTMLImageElement | null>(null);
  const selection = useRef(0);
  const drag = useRef<{ x: number; y: number; panX: number; panY: number } | null>(null);
  const [source, setSource] = useState<HTMLCanvasElement | null>(null);
  const [preview, setPreview] = useState("");
  const [turns, setTurns] = useState(0);
  const [zoom, setZoom] = useState(1);
  const [panX, setPanX] = useState(0);
  const [panY, setPanY] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => () => { selection.current++; }, []);

  const close = useCallback(() => {
    if (busy) return;
    setSource(null);
    setPreview("");
    original.current = null;
    drag.current = null;
    setError("");
  }, [busy]);
  const changeOpen = useCallback((open: boolean) => { if (!open) close(); }, [close]);

  function showImage(image: HTMLImageElement, rotation: number) {
    const prepared = preparePhoto(image, rotation);
    setSource(prepared);
    setPreview(prepared.toDataURL("image/jpeg", 0.9));
    setTurns(rotation);
    setZoom(1);
    setPanX(0);
    setPanY(0);
  }

  async function selectPhoto(file?: File) {
    if (!file) return;
    setError("");
    if (!["image/jpeg", "image/png", "image/webp"].includes(file.type)) {
      setError("Choose a JPG, PNG or WebP photo.");
      return;
    }
    if (file.size > 10 * 1024 * 1024) {
      setError("Choose a photo smaller than 10 MB.");
      return;
    }
    const request = ++selection.current;
    const url = URL.createObjectURL(file);
    try {
      const image = new Image();
      image.src = url;
      await image.decode();
      if (request !== selection.current) return;
      if (image.naturalWidth * image.naturalHeight > 32_000_000) {
        throw new Error("Choose a photo of at most 32 megapixels.");
      }
      showImage(image, 0);
      original.current = image;
    } catch (err) {
      if (request === selection.current) setError(err instanceof Error ? err.message : "The photo could not be opened.");
    } finally {
      URL.revokeObjectURL(url);
    }
  }

  function changed(url: string) {
    onChange(url);
    window.dispatchEvent(new CustomEvent("alias-photo-updated"));
  }

  async function save() {
    if (!source) return;
    setBusy(true);
    setError("");
    try {
      const form = new FormData();
      form.append("photo", await exportPhoto(source, zoom, panX, panY), "profile-photo.jpg");
      const result = await uploadFile(`/api/aliases/${alias.id}/photo`, form) as { avatar_url: string };
      changed(result.avatar_url);
      setSource(null);
      setPreview("");
      original.current = null;
    } catch (err) {
      setError(err instanceof Error ? err.message : "The photo could not be saved.");
    } finally {
      setBusy(false);
    }
  }

  async function remove() {
    setBusy(true);
    setError("");
    try {
      await api.delete(`/api/aliases/${alias.id}/photo`);
      changed("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "The photo could not be removed.");
    } finally {
      setBusy(false);
    }
  }

  const minSide = source ? Math.min(source.width, source.height) : 1;
  const widthPercent = source ? (source.width / minSide) * zoom * 100 : 100;
  const heightPercent = source ? (source.height / minSide) * zoom * 100 : 100;
  const clamp = (value: number) => Math.max(-1, Math.min(1, value));

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-3">
        <AliasAvatar url={alias.avatar_url} address={alias.address} />
        {editable && <>
          <input ref={input} type="file" accept="image/jpeg,image/png,image/webp" className="sr-only" aria-label={`Choose profile photo for ${alias.address}`} tabIndex={-1} onChange={(event) => {
            void selectPhoto(event.target.files?.[0]);
            event.target.value = "";
          }} />
          <Button variant="outline" size="sm" disabled={busy} onClick={() => input.current?.click()}>
            <Camera className="mr-2 h-3.5 w-3.5" />{alias.avatar_url ? "Change photo" : "Upload photo"}
          </Button>
          {alias.avatar_url && <Button variant="ghost" size="sm" disabled={busy} onClick={remove}>Remove photo</Button>}
        </>}
      </div>
      {error && !source && <p role="alert" className="text-xs text-destructive">{error}</p>}
      <Dialog open={!!source} onOpenChange={changeOpen}>
        <DialogContent className="max-w-md max-h-[90dvh] overflow-y-auto" onClose={busy ? undefined : close} aria-labelledby={`photo-title-${alias.id}`}>
          <h2 id={`photo-title-${alias.id}`} className="font-semibold text-lg">Adjust profile photo</h2>
          <p className="mt-1 text-sm text-muted-foreground break-all">{alias.address}</p>
          <p className="mt-2 text-xs text-muted-foreground">Drag the photo or use the sliders to adjust the crop.</p>
          <div className="relative mx-auto my-5 aspect-square w-full max-w-[280px] overflow-hidden bg-muted touch-none cursor-grab active:cursor-grabbing" role="img" aria-label="Circular photo crop preview"
            onPointerDown={(event) => {
              if (busy) return;
              event.currentTarget.setPointerCapture(event.pointerId);
              drag.current = { x: event.clientX, y: event.clientY, panX, panY };
            }}
            onPointerMove={(event) => {
              if (!drag.current || busy) return;
              const size = event.currentTarget.getBoundingClientRect().width;
              const overflowX = size * (widthPercent / 100 - 1) / 2;
              const overflowY = size * (heightPercent / 100 - 1) / 2;
              if (overflowX > 0) setPanX(clamp(drag.current.panX + (event.clientX - drag.current.x) / overflowX));
              if (overflowY > 0) setPanY(clamp(drag.current.panY + (event.clientY - drag.current.y) / overflowY));
            }}
            onPointerUp={() => { drag.current = null; }} onPointerCancel={() => { drag.current = null; }}>
            {preview && <img src={preview} alt="" draggable={false} className="absolute max-w-none select-none pointer-events-none" style={{
              width: `${widthPercent}%`, height: `${heightPercent}%`,
              left: `${50 + panX * (widthPercent - 100) / 2}%`, top: `${50 + panY * (heightPercent - 100) / 2}%`,
              transform: "translate(-50%, -50%)",
            }} />}
            <div className="absolute inset-0 rounded-full border-2 border-white/90 pointer-events-none" style={{ boxShadow: "0 0 0 100px rgba(0,0,0,0.5)" }} />
          </div>
          <div className="space-y-3">
            {([
              ["Zoom", zoom, setZoom, 1, 3, false],
              ["Horizontal position", panX, setPanX, -1, 1, widthPercent <= 100],
              ["Vertical position", panY, setPanY, -1, 1, heightPercent <= 100],
            ] as const).map(([label, value, setter, min, max, unavailable]) => (
              <label key={label} className="flex items-center gap-3 text-xs">
                <span className="w-28 shrink-0">{label}</span>
                <input type="range" aria-label={label} min={min} max={max} step={0.01} value={value} disabled={busy || unavailable} onChange={(event) => setter(Number(event.target.value))} className="w-full accent-primary" />
              </label>
            ))}
          </div>
          <div className="mt-4 flex justify-between gap-2">
            <Button variant="outline" size="sm" disabled={busy} onClick={() => {
              if (original.current) showImage(original.current, (turns + 1) % 4);
            }}><RotateCw className="h-4 w-4 mr-2" />Rotate</Button>
            <Button variant="ghost" size="sm" disabled={busy} onClick={() => { setZoom(1); setPanX(0); setPanY(0); }}>Reset crop</Button>
          </div>
          <p className="mt-4 text-xs text-muted-foreground">This photo will be public and included in emails sent from this alias. Inbox avatars depend on the recipient&apos;s email service.</p>
          {error && <p role="alert" className="mt-3 text-sm text-destructive">{error}</p>}
          <div className="mt-5 flex justify-end gap-2">
            <Button variant="outline" disabled={busy} onClick={close}>Cancel</Button>
            <Button disabled={busy} onClick={save}>{busy && <Spinner className="mr-2" />}Save photo</Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
