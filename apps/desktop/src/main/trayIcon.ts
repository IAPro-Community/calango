import { nativeImage, type NativeImage } from 'electron';

// 32x32 tray icon: yellow rounded square, green bird. Kept as a module-level
// SVG string so the PNG fallback uses the same artwork.
const TRAY_SVG =
  '<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 32 32"><rect width="32" height="32" rx="7" fill="#FDD838"/><path d="M7 21c2-3 5-4 8-4 1-4 4-7 9-7l3 2-3 2c1 2 2 4 3 6l-3 2-3-3-5 1-3 5-3-1 2-5-4 2z" fill="#42912E" stroke="#22180F" stroke-width="1.2" stroke-linejoin="round"/><circle cx="23" cy="12" r="1" fill="#22180F"/></svg>';

// Electron renders SVG data URLs on Linux but not on Windows and macOS, where
// Tray needs a raster image. The tray therefore uses an explicit PNG asset and
// keeps the SVG only as a last resort for development runs without one.
export const TRAY_ICON_PNG = 'tray.png';

export interface TrayIconSources {
  /** Absolute path to tray.png shipped next to the compiled main bundle. */
  pngPath: string;
}

function svgImage(): NativeImage {
  return nativeImage.createFromDataURL(`data:image/svg+xml;charset=utf-8,${encodeURIComponent(TRAY_SVG)}`);
}

// loadTrayIcon returns a raster tray image. A missing or empty PNG returns null
// so the caller can say so instead of creating an invisible tray.
export function loadTrayIcon(sources: TrayIconSources): NativeImage | null {
  const image = nativeImage.createFromPath(sources.pngPath);
  if (image.isEmpty()) {
    return null;
  }
  return image;
}

// fallbackTrayIcon is only for development when the PNG asset is absent.
export function fallbackTrayIcon(): NativeImage {
  return svgImage();
}
