export function getPhotoCrop(width: number, height: number, zoom: number, panX: number, panY: number) {
  const size = Math.min(width, height) / Math.max(1, zoom);
  const clamp = (value: number) => Math.max(-1, Math.min(1, value));
  return {
    x: ((width - size) * (1 - clamp(panX))) / 2,
    y: ((height - size) * (1 - clamp(panY))) / 2,
    size,
  };
}

// Normalize orientation and bound the editing canvas before making previews.
export function preparePhoto(image: HTMLImageElement, turns: number) {
  const scale = Math.min(1, 2048 / Math.max(image.naturalWidth, image.naturalHeight));
  const width = Math.round(image.naturalWidth * scale);
  const height = Math.round(image.naturalHeight * scale);
  const canvas = document.createElement("canvas");
  const sideways = turns % 2 !== 0;
  canvas.width = sideways ? height : width;
  canvas.height = sideways ? width : height;
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("Your browser could not prepare the photo");
  ctx.fillStyle = "#fff";
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  ctx.translate(canvas.width / 2, canvas.height / 2);
  ctx.rotate((turns * Math.PI) / 2);
  ctx.drawImage(image, -width / 2, -height / 2, width, height);
  return canvas;
}

export async function exportPhoto(source: HTMLCanvasElement, zoom: number, panX: number, panY: number): Promise<Blob> {
  const crop = getPhotoCrop(source.width, source.height, zoom, panX, panY);
  const canvas = document.createElement("canvas");
  canvas.width = canvas.height = 512;
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("Your browser could not crop the photo");
  ctx.imageSmoothingEnabled = true;
  ctx.imageSmoothingQuality = "high";
  ctx.drawImage(source, crop.x, crop.y, crop.size, crop.size, 0, 0, 512, 512);
  return new Promise((resolve, reject) => canvas.toBlob(
    (blob) => blob ? resolve(blob) : reject(new Error("Your browser could not export the photo")),
    "image/jpeg", 0.9,
  ));
}
