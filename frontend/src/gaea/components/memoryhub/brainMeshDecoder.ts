// Shared decoder for the NIH 3D brain mesh embedded in brainMeshData.ts.
// Keeping this module lazy-loadable lets voice and memory surfaces share the
// exact same topology without paying the 500 KB payload on every screen.
import { BRAIN_MESH_B64 } from "./brainMeshData";

export interface BrainMeshData {
  pos: Float32Array;
  shade: Float32Array;
  fibers: number[][];
  tri: Uint16Array;
}

let cachedMesh: BrainMeshData | null = null;

export function decodeBrainMesh(): BrainMeshData {
  if (cachedMesh) return cachedMesh;
  const bin = atob(BRAIN_MESH_B64);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  const dv = new DataView(bytes.buffer);
  let offset = 0;
  const magic = dv.getUint32(offset, true);
  offset += 4;
  const vertexCount = dv.getUint32(offset, true);
  offset += 4;
  const fiberCount = dv.getUint32(offset, true);
  offset += 4;
  const triangleCount = dv.getUint32(offset, true);
  offset += 4;
  if (magic !== 0x314e5242) throw new Error("bad brain mesh data");

  const pos = new Float32Array(vertexCount * 3);
  const shade = new Float32Array(vertexCount);
  for (let i = 0; i < vertexCount; i += 1) {
    pos[i * 3] = dv.getInt16(offset, true) / 28000;
    pos[i * 3 + 1] = dv.getInt16(offset + 2, true) / 28000;
    pos[i * 3 + 2] = dv.getInt16(offset + 4, true) / 28000;
    shade[i] = dv.getUint16(offset + 6, true) / 1000;
    offset += 8;
  }

  const fibers: number[][] = [];
  for (let fiber = 0; fiber < fiberCount; fiber += 1) {
    const length = dv.getUint16(offset, true);
    offset += 2;
    const indices = new Array<number>(length);
    for (let i = 0; i < length; i += 1) {
      indices[i] = dv.getUint16(offset, true);
      offset += 2;
    }
    fibers.push(indices);
  }

  const tri = new Uint16Array(triangleCount * 3);
  for (let i = 0; i < triangleCount * 3; i += 1) {
    tri[i] = dv.getUint16(offset, true);
    offset += 2;
  }

  cachedMesh = { pos, shade, fibers, tri };
  return cachedMesh;
}
