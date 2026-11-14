import { describe, expect, it } from 'vitest';
import {
  LOCAL_ENGINE_WARN_LIMIT,
  countLocalEngines,
  exceedsLocalEngineLimit,
} from './telemetryOverload';

describe('telemetry local engine overload policy', () => {
  it('allows up to six local engines and warns only above six', () => {
    expect(LOCAL_ENGINE_WARN_LIMIT).toBe(6);
    expect(exceedsLocalEngineLimit(4)).toBe(false);
    expect(exceedsLocalEngineLimit(6)).toBe(false);
    expect(exceedsLocalEngineLimit(7)).toBe(true);
  });

  it('counts local engines plus a running ComfyUI without counting cloud engines', () => {
    const engines = [
      { isLocal: true },
      { isLocal: true },
      { isLocal: false },
      {},
    ];
    expect(countLocalEngines(engines, false)).toBe(2);
    expect(countLocalEngines(engines, true)).toBe(3);
  });
});
