import qrcode from 'qrcode-generator';

// The pairing QR is generated locally; no external service sees the code.

export interface QrMatrix {
  /** Modules per side, without the quiet zone. */
  size: number;
  /** SVG path of the dark modules, in module units, offset by the quiet zone. */
  path: string;
  /** Modules per side including the quiet zone; the SVG viewBox side. */
  viewBox: number;
}

/** Quiet zone of four modules, as the QR specification requires. */
export const QR_QUIET_ZONE = 4;

/** qrMatrix encodes text (byte mode, error correction M, smallest fitting version). */
export function qrMatrix(text: string): QrMatrix {
  const qr = qrcode(0, 'M');
  qr.addData(text, 'Byte');
  qr.make();
  const size = qr.getModuleCount();
  const parts: string[] = [];
  for (let row = 0; row < size; row += 1) {
    for (let col = 0; col < size; col += 1) {
      if (qr.isDark(row, col)) {
        parts.push(`M${col + QR_QUIET_ZONE} ${row + QR_QUIET_ZONE}h1v1h-1z`);
      }
    }
  }
  return { size, path: parts.join(''), viewBox: size + QR_QUIET_ZONE * 2 };
}
