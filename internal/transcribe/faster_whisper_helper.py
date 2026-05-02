import argparse
import json
import sys

from faster_whisper import WhisperModel


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Transcribe audio with faster-whisper")
    parser.add_argument("--audio", required=True)
    parser.add_argument("--model", required=True)
    parser.add_argument("--device", default="cpu")
    parser.add_argument("--compute-type", default="int8")
    parser.add_argument("--language", default="")
    return parser.parse_args()


def main() -> int:
    args = parse_args()

    model = WhisperModel(args.model, device=args.device, compute_type=args.compute_type)
    kwargs = {"beam_size": 1}
    if args.language:
        kwargs["language"] = args.language

    segments, _info = model.transcribe(args.audio, **kwargs)
    text = "".join(segment.text for segment in segments).strip()
    json.dump({"text": text}, sys.stdout)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
