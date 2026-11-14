/** Local model engines are allowed up to six before the shell warns. */
export const LOCAL_ENGINE_WARN_LIMIT = 6;

export interface LocalEngineTelemetry {
  engine: string;
  model?: string;
  isLocal?: boolean;
  connected?: boolean;
}

function runningLocalEngines(
  engines: ReadonlyArray<LocalEngineTelemetry>,
): LocalEngineTelemetry[] {
  return engines.filter((engine) => engine.isLocal && engine.connected === true);
}

export function countLocalEngines(
  engines: ReadonlyArray<LocalEngineTelemetry>,
  comfyRunning?: boolean,
): number {
  return runningLocalEngines(engines).length + (comfyRunning ? 1 : 0);
}

export function runningLocalEngineLabels(
  engines: ReadonlyArray<LocalEngineTelemetry>,
): string[] {
  return runningLocalEngines(engines).map((engine) => (
    `${engine.engine}${engine.model ? `·${engine.model.split('/').pop()}` : ''}`
  ));
}

export function exceedsLocalEngineLimit(count: number): boolean {
  return count > LOCAL_ENGINE_WARN_LIMIT;
}
