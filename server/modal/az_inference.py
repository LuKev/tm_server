"""Deploy the Terra Mystica neural evaluator as a Modal web service."""

import os
import subprocess

import modal


MODEL_ID = os.environ.get(
    "TM_AZ_MODEL_ID", "promoted-h512-replay-localproof-20260713"
)
MODEL_ROOT = "/models"
MODEL_PATH = f"{MODEL_ROOT}/{MODEL_ID}/model.pt"

app = modal.App("tm-az-inference")
models = modal.Volume.from_name("tm-az-models", create_if_missing=False)
image = (
    modal.Image.debian_slim(python_version="3.11")
    .run_commands(
        "pip install --index-url https://download.pytorch.org/whl/cpu torch==2.8.0"
    )
    .add_local_file(
        "cmd/az_infer_torch/az_infer_torch.py",
        remote_path="/app/az_infer_torch.py",
    )
)


@app.function(
    image=image,
    cpu=2.0,
    memory=2048,
    min_containers=1,
    timeout=24 * 60 * 60,
    volumes={MODEL_ROOT: models},
)
@modal.web_server(port=9097, startup_timeout=120)
def serve():
    if not os.path.isfile(MODEL_PATH):
        raise RuntimeError(f"checkpoint not found: {MODEL_PATH}")
    subprocess.Popen(
        [
            "python",
            "/app/az_infer_torch.py",
            f"--checkpoint={MODEL_PATH}",
            "--host=0.0.0.0",
            "--port=9097",
            "--torch-threads=2",
            "--torch-interop-threads=1",
        ]
    )
