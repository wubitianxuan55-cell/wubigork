from __future__ import annotations

import unittest
from types import SimpleNamespace

import gaea_sidecar_headers as headers


class GaeaSidecarHeadersTest(unittest.TestCase):
    def test_last_user_item_id_uses_latest_protocol_item(self) -> None:
        active_chat = SimpleNamespace(
            buffer=[
                SimpleNamespace(role="user", id="msg_old"),
                SimpleNamespace(role="assistant", id="msg_answer"),
                SimpleNamespace(role="user", id="msg_current"),
            ]
        )
        self.assertEqual(headers._last_user_item_id(active_chat), "msg_current")

    def test_chat_request_receives_turn_headers_without_mutating_text(self) -> None:
        headers.install()
        from speech_to_speech.LLM.chat_completions_language_model import (
            ChatCompletionsApiModelHandler,
        )

        captured = {}

        class Completions:
            @staticmethod
            def create(**kwargs):
                captured.update(kwargs)
                return object()

        handler = object.__new__(ChatCompletionsApiModelHandler)
        handler.client = SimpleNamespace(chat=SimpleNamespace(completions=Completions()))
        handler.model_name = "qwen3-4b"
        handler.stream = True
        handler.request_timeout = 3.0
        handler._extra_body = None

        messages = [{"role": "user", "content": "你好"}]
        token = headers._CURRENT_TURN.set(
            headers.VoiceTurnHeaders(turn_id="turn_7", turn_revision=2, item_id="msg_7")
        )
        try:
            handler._request(messages, {})
        finally:
            headers._CURRENT_TURN.reset(token)

        self.assertEqual(captured["messages"], messages)
        self.assertEqual(
            captured["extra_headers"],
            {
                "X-Gaea-Voice-Turn-Id": "turn_7",
                "X-Gaea-Voice-Turn-Revision": "2",
                "X-Gaea-Voice-Item-Id": "msg_7",
            },
        )

    def test_existing_extra_headers_are_preserved(self) -> None:
        existing = {"X-Provider-Trace": "keep"}
        merged = headers._merge_headers(existing, {"X-Gaea-Voice-Turn-Id": "turn_8"})
        self.assertEqual(
            merged,
            {
                "X-Provider-Trace": "keep",
                "X-Gaea-Voice-Turn-Id": "turn_8",
            },
        )


if __name__ == "__main__":
    unittest.main()
