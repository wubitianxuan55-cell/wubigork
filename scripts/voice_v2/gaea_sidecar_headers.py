"""Forward Gaea voice turn identity through the speech-to-speech LLM call.

The sidecar speaks a standard OpenAI Realtime protocol.  Its public response
events identify input and output items, but the nested Chat Completions call
does not expose the input item id by default.  Gaea needs that id to bind the
generation-time persona snapshot to the later authoritative commit.

This module installs a narrow, idempotent monkeypatch at sidecar startup.  It
does not alter prompts or assistant text: it only adds three private request
headers to the Chat Completions call consumed by Gaea's localhost gateway.
"""

from __future__ import annotations

from collections.abc import Mapping
from contextvars import ContextVar
from dataclasses import dataclass
from typing import Any


@dataclass(frozen=True)
class VoiceTurnHeaders:
    """Stable sidecar identity for one generation."""

    turn_id: str
    turn_revision: int | None
    item_id: str

    def as_http_headers(self) -> dict[str, str]:
        headers = {"X-Gaea-Voice-Turn-Id": self.turn_id}
        if self.turn_revision is not None:
            headers["X-Gaea-Voice-Turn-Revision"] = str(self.turn_revision)
        if self.item_id:
            headers["X-Gaea-Voice-Item-Id"] = self.item_id
        return headers


# The provider request is made synchronously from inside the generator that
# owns this context. ContextVar keeps concurrent sidecar sessions isolated.
_CURRENT_TURN: ContextVar[VoiceTurnHeaders | None] = ContextVar(
    "gaea_voice_turn_headers",
    default=None,
)


def _clean_id(value: Any) -> str:
    if not isinstance(value, str):
        return ""
    return value.strip()


def _last_user_item_id(active_chat: Any) -> str:
    """Return the protocol item id of the newest user turn in the snapshot."""

    buffer = getattr(active_chat, "buffer", None)
    if not isinstance(buffer, list):
        return ""
    for item in reversed(buffer):
        if getattr(item, "role", None) != "user":
            continue
        item_id = _clean_id(getattr(item, "id", None))
        if item_id:
            return item_id
    return ""


def _merge_headers(existing: Any, extra: Mapping[str, str]) -> dict[str, str] | None:
    if existing is None:
        return dict(extra)
    if not isinstance(existing, Mapping):
        # An unexpected non-mapping header type belongs to the upstream package;
        # preserve it rather than silently replacing provider configuration.
        return None
    merged = {str(key): str(value) for key, value in existing.items()}
    merged.update(extra)
    return merged


def install() -> bool:
    """Install the request metadata bridge; safe to call more than once."""

    from speech_to_speech.LLM.base_openai_compatible_language_model import (
        BaseOpenAICompatibleHandler,
    )
    from speech_to_speech.LLM.chat_completions_language_model import (
        ChatCompletionsApiModelHandler,
    )

    if getattr(ChatCompletionsApiModelHandler, "_gaea_turn_header_patch", False):
        return False

    original_generate = BaseOpenAICompatibleHandler._generate
    original_request = ChatCompletionsApiModelHandler._request

    def generate_with_turn_headers(
        self: Any,
        active_chat: Any,
        original_chat: Any,
        turn: Any,
        optional_kwargs: dict[str, Any],
        *args: Any,
        **kwargs: Any,
    ):
        input_history = kwargs.get("input_history")
        item_id = _clean_id(getattr(input_history, "id", None)) or _last_user_item_id(active_chat)
        turn_id = _clean_id(getattr(turn, "turn_id", None))
        revision = getattr(turn, "turn_revision", None)
        revision = revision if isinstance(revision, int) else None
        metadata = VoiceTurnHeaders(turn_id=turn_id, turn_revision=revision, item_id=item_id) if turn_id else None
        token = _CURRENT_TURN.set(metadata)
        try:
            yield from original_generate(
                self,
                active_chat,
                original_chat,
                turn,
                optional_kwargs,
                *args,
                **kwargs,
            )
        finally:
            _CURRENT_TURN.reset(token)

    def request_with_turn_headers(
        self: Any,
        api_input: list[dict[str, Any]],
        optional_kwargs: dict[str, Any],
    ):
        metadata = _CURRENT_TURN.get()
        if metadata is None:
            return original_request(self, api_input, optional_kwargs)
        request_kwargs = dict(optional_kwargs or {})
        merged = _merge_headers(request_kwargs.get("extra_headers"), metadata.as_http_headers())
        if merged is not None:
            request_kwargs["extra_headers"] = merged
        return original_request(self, api_input, request_kwargs)

    BaseOpenAICompatibleHandler._generate = generate_with_turn_headers
    ChatCompletionsApiModelHandler._request = request_with_turn_headers
    ChatCompletionsApiModelHandler._gaea_turn_header_patch = True
    print("[gaea-voice-v2] turn correlation headers enabled", flush=True)
    return True
