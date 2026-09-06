"""Fail-closed bridge to the same Go simulation compiled into the browser."""
import json
import os
import subprocess
from functools import lru_cache
from threading import BoundedSemaphore

from fastapi import HTTPException

from app.schemas import RunFields

VERIFIER_PATH = os.getenv("VERIFIER_PATH", "/usr/local/bin/rogue-verifier")
_slots = BoundedSemaphore(2)


@lru_cache(maxsize=1)
def rules_version():
    result = subprocess.run([VERIFIER_PATH, "--version"], capture_output=True,
                            text=True, timeout=2, check=True)
    return result.stdout.strip()


def verify(seed: str, actions: str) -> dict:
    if not _slots.acquire(blocking=False):
        raise HTTPException(503, "Verification busy; retry the same run later")
    try:
        result = subprocess.run(
            [VERIFIER_PATH], input=json.dumps({"seed": int(seed), "actions": actions}),
            capture_output=True, text=True, timeout=3,
            env={**os.environ, "GOMAXPROCS": "1", "GOMEMLIMIT": "64MiB"},
        )
        if result.returncode == 2:
            raise HTTPException(422, "Invalid or unfinished replay")
        if result.returncode != 0:
            raise HTTPException(503, "Verification unavailable")
        # Validate even trusted subprocess output before writing SQL INTEGERs.
        return RunFields(**json.loads(result.stdout)).model_dump(exclude={"player_name"})
    except subprocess.TimeoutExpired:
        raise HTTPException(422, "Replay exceeded the verification time limit") from None
    except (OSError, ValueError):
        raise HTTPException(503, "Verification unavailable") from None
    finally:
        _slots.release()
