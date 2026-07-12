# Modal Neural Inference

The model package is backed up under Google Drive:

`Terra Mystica AI/models/<model-id>/`

Modal uses a runtime mirror because Google Drive is not a mounted production
filesystem. Upload the exact hash-verified checkpoint from the Drive package or
its matching local package:

```bash
modal volume create tm-az-models
modal volume put tm-az-models \
  ../artifacts/az/models/<model-id> \
  /<model-id>
modal deploy modal/az_inference.py
```

Set `TM_AZ_MODEL_ID` while deploying to pin a different package. Configure the
website backend with the deployed URL:

```text
TM_AZ_MODEL_URL=https://<modal-host>/evaluate
TM_AZ_REQUIRE_NEURAL=true
```

Verify `/healthz` on Modal and `/api/ai/status` on the website backend before
starting a model game.
