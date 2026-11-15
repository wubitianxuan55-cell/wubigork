export interface ProgrammingWebStatus {
  running: boolean
  owned: boolean
  starting: boolean
  pid: number
  url: string
  root: string
  home: string
  version: string
  log: string
  uptime_s: number
}

export interface ProgrammingWebPreflight {
  node_found: boolean
  node_path: string
  install_ready: boolean
  profile_ready: boolean
  plugin_ready: boolean
  rea_ready: boolean
  isolated: boolean
  version_ok: boolean
  port_free: boolean
  all_ready: boolean
  root: string
  home: string
  agents_home: string
  base: string
  version: string
  expected: string
}

export interface ProgrammingWebLogTail {
  exists: boolean
  path: string
  lines: string[]
  error: string
}
