import os
import json
import shutil
import tempfile
import subprocess
from pathlib import Path
from typing import Optional

from fastapi import FastAPI, File, Form, HTTPException, UploadFile
from fastapi.responses import JSONResponse

MSS_BIN = Path(__file__).resolve().parent / "bin" / "mss"

app = FastAPI(
    title="MoonShort Script API",
    description="Compile and decompile MoonShort Script (MSS) files via HTTP.",
    version="1.0.0",
)


def _run_mss(*args: str, workdir: Optional[str] = None, timeout: int = 30) -> subprocess.CompletedProcess:
    cmd = [str(MSS_BIN), *args]
    return subprocess.run(
        cmd,
        capture_output=True,
        text=True,
        cwd=workdir,
        timeout=timeout,
    )


@app.post("/compile")
async def compile_script(
    script: UploadFile = File(..., description="MSS script file (.md)"),
    assets: Optional[UploadFile] = File(default=None, description="Optional assets mapping JSON file"),
):
    """
    Compile an MSS script (.md) into structured JSON.

    Returns the compiled episode JSON. If an assets mapping is provided,
    asset semantic names are resolved to full URLs.
    """
    tmpdir = tempfile.mkdtemp(prefix="mss_compile_")
    try:
        script_bytes = await script.read()
        script_text = script_bytes.decode("utf-8")
        script_path = os.path.join(tmpdir, "script.md")
        with open(script_path, "w", encoding="utf-8") as f:
            f.write(script_text)

        args = ["compile", script_path, "-o", os.path.join(tmpdir, "output.json")]

        if assets is not None:
            assets_bytes = await assets.read()
            assets_text = assets_bytes.decode("utf-8")
            assets_path = os.path.join(tmpdir, "assets.json")
            with open(assets_path, "w", encoding="utf-8") as f:
                f.write(assets_text)
            args.insert(2, "--assets")
            args.insert(3, assets_path)

        proc = _run_mss(*args, timeout=30)

        if proc.returncode != 0:
            raise HTTPException(status_code=422, detail={"error": proc.stderr.strip()})

        output_path = os.path.join(tmpdir, "output.json")
        with open(output_path, "r", encoding="utf-8") as f:
            result = json.load(f)

        return JSONResponse(content=result)

    except subprocess.TimeoutExpired:
        raise HTTPException(status_code=504, detail="Compilation timed out")
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
    finally:
        shutil.rmtree(tmpdir, ignore_errors=True)


@app.post("/decompile")
async def decompile_json(
    compiled: UploadFile = File(..., description="Compiled MSS JSON file"),
):
    """
    Decompile compiled MSS JSON back into MSS script and asset mapping.

    Returns the reconstructed MSS source (.md) and the recovered asset mapping.
    """
    tmpdir = tempfile.mkdtemp(prefix="mss_decompile_")
    try:
        compiled_bytes = await compiled.read()
        compiled_text = compiled_bytes.decode("utf-8")
        input_path = os.path.join(tmpdir, "input.json")
        with open(input_path, "w", encoding="utf-8") as f:
            f.write(compiled_text)

        output_dir = os.path.join(tmpdir, "decompiled")

        proc = _run_mss("decompile", input_path, "-o", output_dir, timeout=30)

        warnings = []
        if proc.stderr.strip():
            for line in proc.stderr.strip().split("\n"):
                line = line.strip()
                if line.startswith("warning:"):
                    warnings.append(line.removeprefix("warning:").strip())
                elif line.startswith("wrote"):
                    pass
                elif line:
                    warnings.append(line)

        if not os.path.isdir(output_dir):
            raise HTTPException(
                status_code=422,
                detail={"error": proc.stderr.strip() or "Decompilation produced no output"},
            )

        mss_files = {}
        mapping = None
        for fname in os.listdir(output_dir):
            fpath = os.path.join(output_dir, fname)
            if fname.endswith(".md") or fname.endswith(".mss.md"):
                with open(fpath, "r", encoding="utf-8") as f:
                    mss_files[fname] = f.read()
            elif fname.endswith(".json"):
                with open(fpath, "r", encoding="utf-8") as f:
                    mapping = json.load(f)

        return JSONResponse(
            content={
                "episodes": mss_files,
                "asset_mapping": mapping,
                "warnings": warnings,
            }
        )

    except subprocess.TimeoutExpired:
        raise HTTPException(status_code=504, detail="Decompilation timed out")
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
    finally:
        shutil.rmtree(tmpdir, ignore_errors=True)


@app.get("/health")
async def health():
    """Health check endpoint."""
    if not MSS_BIN.exists():
        return JSONResponse(
            status_code=503,
            content={"status": "unhealthy", "reason": "mss binary not found"},
        )
    return {"status": "ok"}
