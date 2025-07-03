
import os
import platform


def default_nvidia_cudnn():
    match platform.platform() :
        case "windows": return "C:\\Program Files\\NVIDIA\\CUDNN\\v9.10\\bin\\12.9"
        case "linux" : return "C:\\Program Files\\NVIDIA\\CUDNN\\v9.10\\bin\\12.9"
        case _: return ""

# Cuda specif path to CUDNN
NVIDIA_CUDNN = os.getenv("NVIDIA_CUDNN") or default_nvidia_cudnn()

# Regular multilingual models (99 languages)
LOW_END_DEVICES = ["tiny", "base"]
MED_RANGE_DEVICES = ["small", "medium"]
HIGH_END_DEVICES = ["large-v1", "large-v2", "large-v3"]

# English-only Distil-Whisper models
LOW_END_DEVICES_EN = ["distil-small.en"]
MED_RANGE_DEVICES_EN = ["distil-medium.en"]
HIGH_END_DEVICES_EN = ["distil-large-v2"]