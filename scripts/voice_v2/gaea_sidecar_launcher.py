"""Launch speech-to-speech with Gaea's turn-correlation metadata patch."""

from __future__ import annotations

from gaea_sidecar_headers import install


def main() -> None:
    install()
    from speech_to_speech.cli import main as cli_main

    cli_main()


if __name__ == "__main__":
    main()
