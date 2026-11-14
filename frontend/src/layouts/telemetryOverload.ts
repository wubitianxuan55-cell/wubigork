/** Local model engines are allowed up to six before the shell warns. */
export const LOCAL_ENGINE_WARN_LIMIT = 6;

export function countLocalEngines(
  engines: ReadonlyArray<{ isLocal?: boolean }>,
  comfyRunning?: boolean,
): number {
  return engines.filter((engine) => engine.isLocal).length + (comfyRunning ? 1 : 0);
}

export function exceedsLocalEngineLimit(count: number): boolean {
  return count > LOCAL_ENGINE_WARN_LIMIT;
}
