import React from "react";
import { MemoryAtlasStage } from "../../gaea/components/memoryhub/MemoryAtlasStage";
import type { GraphLink, GraphNode } from "../../gaea/lib/types";
import type { VoiceSessionState } from "../session/voiceSession";

const EMPTY_NODES: GraphNode[] = [];
const EMPTY_LINKS: GraphLink[] = [];

export interface VoiceMemoryBrainLayerProps {
  phase: VoiceSessionState["phase"];
  connected: boolean;
  micLevelRef?: { current: number };
  playLevelRef?: { current: number };
  onReady?: () => void;
}

/**
 * Voice-specialised adapter around the memory-atlas brain. It keeps the exact
 * cortical mesh and fiber layers, but adds a phase/audio pulse channel instead
 * of memory-node interaction.
 */
const VoiceMemoryBrainLayer: React.FC<VoiceMemoryBrainLayerProps> = ({
  phase,
  connected,
  micLevelRef,
  playLevelRef,
  onReady,
}) => (
  <MemoryAtlasStage
    nodes={EMPTY_NODES}
    links={EMPTY_LINKS}
    paused={false}
    autoRotate
    showRings
    showDust
    voicePhase={phase}
    voiceConnected={connected}
    voiceLevelRef={micLevelRef}
    voicePlayLevelRef={playLevelRef}
    onReady={onReady}
  />
);

export default VoiceMemoryBrainLayer;
