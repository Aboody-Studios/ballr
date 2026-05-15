#!/usr/bin/env python3
"""Ballr CV Pipeline - CLI entry point.

Usage:
    python3 main.py --video /path/to/video.mp4 --match-id m-123 --user-id u-456 --shirt-number 10 --position CM [--output /path/to/output.json]
"""

import argparse
import json
import logging
import sys


def setup_logging():
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
        stream=sys.stderr,
    )


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Ballr CV Pipeline")
    parser.add_argument("--video", required=True, help="path to video file")
    parser.add_argument("--match-id", required=True, help="match UUID")
    parser.add_argument("--user-id", required=True, help="user UUID")
    parser.add_argument("--shirt-number", required=True, type=int, help="player shirt number")
    parser.add_argument("--position", required=True, help="player position (e.g., CM, ST, GK)")
    parser.add_argument("--output", default=None, help="output JSON file path (default: stdout)")
    return parser.parse_args(argv)


def main() -> None:
    setup_logging()
    logger = logging.getLogger("main")
    args = parse_args(sys.argv[1:])
    logger.info("starting cv pipeline: match=%s, user=%s, shirt=%d, pos=%s",
                args.match_id, args.user_id, args.shirt_number, args.position)

    try:
        from pipeline import run_pipeline

        result = run_pipeline(
            video_path=args.video,
            match_id=args.match_id,
            user_id=args.user_id,
            shirt_number=args.shirt_number,
            position=args.position,
        )

        json_str = json.dumps(result, indent=2, default=str)

        if args.output:
            with open(args.output, "w") as f:
                f.write(json_str)
            logger.info("output written to %s", args.output)
        else:
            print(json_str)

    except FileNotFoundError as e:
        logger.error("file error: %s", e)
        print(f"ERROR: {e}", file=sys.stderr)
        sys.exit(1)
    except ValueError as e:
        logger.error("value error: %s", e)
        print(f"ERROR: {e}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        logger.error("pipeline failed: %s", e, exc_info=True)
        print(f"ERROR: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
