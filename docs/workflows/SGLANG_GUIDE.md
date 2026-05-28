# 🚀 Musu SGLang Operations Guide (RTX 5070 Ti Optimization)

This guide details how to leverage your **RTX 5070 Ti 16GB (Blackwell)** for professional-grade AI research and marketing using SGLang.

## 🛠️ 1. Infrastructure Setup (WSL2)

SGLang requires a Linux environment. On Windows, use **WSL2 (Ubuntu 22.04+)**.

### A. Prerequisites
1.  **NVIDIA Drivers:** Ensure latest Windows drivers are installed.
2.  **Docker Desktop:** Enable "WSL2 Based Engine".
3.  **NVIDIA Container Toolkit:** [Install guide](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html).

### B. Launch SGLang (Docker)
Run this command in your WSL2 terminal to launch an optimized Llama-3.1-8B server:

```bash
docker run --gpus all \
  --shm-size 16g \
  -p 30000:30000 \
  -v ~/.cache/huggingface:/root/.cache/huggingface \
  --ipc=host \
  lmsysorg/sglang:latest \
  python3 -m sglang.launch_server \
    --model-path meta-llama/Llama-3.1-8B-Instruct \
    --host 0.0.0.0 \
    --port 30000 \
    --mem-fraction-static 0.8
```

*Note: `--mem-fraction-static 0.8` ensures enough VRAM is left for OS and browser activities on your 16GB card.*

---

## 🏗️ 2. Musu Integration

The Musu ecosystem is now **Engine Agnostic**. You can switch to SGLang using CLI flags or config.

### A. Global Configuration
Edit `projects/default/config.yaml` (or your project config):

```yaml
ai_provider: "sglang"
ai_url: "http://localhost:30000/v1" # SGLang Default
```

### B. CLI Override
```bash
./musu-crawl research "AI Marketing Trends" --ai-url http://localhost:30000/v1
```

---

## 🏎️ 3. Why this matters for your 5070 Ti

1.  **RadixAttention:** Your Marketing Bible and system prompts are now cached in VRAM. First-token latency is effectively eliminated.
2.  **FSM Decoding:** JSON responses for `--json` mode are generated at the hardware limit of your tensor cores.
3.  **Agentic Speed:** Parallel agents (Copywriter + Critic) can now run without sequential bottlenecks.

---
**Status:** Musu Universal AI Gateway is [PASS]
**Ready for Deployment.**
