import defautls

def check_cuda_available():

    try:
        import torch
        if torch.cuda.is_available():
            return True
    except ImportError:
        pass

def nvidia_msi_check():
    try:
        import subprocess
        subprocess.check_output(["nvidia-smi"], stderr=subprocess.STDOUT)
        return True
    except:
        pass

def has_cudart_dll():

    DEFAULT_NVIDIA_CUDNN = defautls.NVIDIA_CUDNN

    if len(DEFAULT_NVIDIA_CUDNN) == "":
        return False, "couldn't load the default nvidia cudnn path"

    try:
        import os,glob
        os_env_paths = os.environ.get("Path") or ""
        if len(os_env_paths) == 0:
            return False , "Machine with no system path, machine is dead"
        path_list = os_env_paths.split(";")
        if path_list.index(DEFAULT_NVIDIA_CUDNN):
            files = glob.glob(os.path.join(DEFAULT_NVIDIA_CUDNN + os.path.sep, "cudnn_*.dll"))
            return len(files) > 0 , ""
        return False, "couldn't find the nvidia cudnn path"
    except:
        return False, "exception error, please retry, if the error persist, check https://..."