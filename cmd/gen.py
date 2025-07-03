# std modules
from argparse import ArgumentParser
import os
import warnings

if os.getenv("WORK_ENV") == "PRODUCTION":
    warnings.filterwarnings(
        "ignore",
        message="pkg_resources is deprecated as an API.*",
        category=UserWarning,
        module="ctranslate2.*"
    )

from faster_whisper import WhisperModel

# dev modules
from cli_utils import Logger
import cuda_check
import defautls

def format_srt_timestamp(seconds:float) -> str:
    h, remainder = divmod(seconds, 3600)
    m, s = divmod(remainder, 60)
    ms = int((s % 1) * 1000)
    return f"{int(h):02d}:{int(m):02d}:{int(s):02d}.{ms:03d}"

def char_parsing_correct_format(text: str) -> str:
    """Optimized HTML entity replacement using str.translate()"""
    # Create translation table once (much faster than repeated replacements)
    translation_table = str.maketrans({
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&apos;',
        '–': '&ndash;',
        '—': '&mdash;',
        '©': '&copy;',
        '®': '&reg;',
        '™': '&trade;',
        '≈': '&asymp;',
        '£': '&pound;',
        '€': '&euro;',
        '°': '&deg;'
    })
    return text.translate(translation_table)


def gen_sub_command(global_parser: ArgumentParser) -> None:
    gen_sub_command = global_parser.add_subparsers(dest="command")

    gen_sub_parser = gen_sub_command.add_parser("gen",help="generates the vtt file of audio or video")

    gen_sub_parser.add_argument('audio_file', help='Audio file to transcribe')
    gen_sub_parser.add_argument('-m', '--model', default='base', help='Model size')
    gen_sub_parser.add_argument('-l', '--language', default='en', help='Language code')
    gen_sub_parser.add_argument('-o', '--output_file' , help='Output file with format (vtt/srt)')
    gen_sub_parser.add_argument('--device', default='cuda', help='Device (cpu/cuda)')

    args = global_parser.parse_args()
    error = ""

    args_config = {
        "audio_file": args.audio_file,
        "model" : args.model,
        "language": args.language,
        "output_file" : args.output_file,
        "device" : args.device
    }

    logger = Logger()

    cuda_available = cuda_check.check_cuda_available()

    if cuda_available is None:
            cuda_available = cuda_check.nvidia_msi_check()

    if cuda_available is None:
            cuda_available, error = cuda_check.has_cudart_dll()

    if cuda_available and len(error) == 0:
        args_config["device"] = "cuda"
    else:

        if len(error) > 0 :
            return logger.Failure("ERROR:", [f"{error}"])

        args.device = "cpu"
        if defautls.HIGH_END_DEVICES_EN.count(args.model) > 0 or defautls.HIGH_END_DEVICES.count(args.model) > 0:
            logger.Warning("WARNING:",["High-end models not recommended. May cause instability on lower-end hardware, use at your own risk"])
            choice = input("Use base model for faster results? [y/n]: ").strip().lower()
            if choice == "n":
                args_config["model"] = "base"

    model = WhisperModel(args_config["model"], device=args_config["device"])

    print(f"Transcribing {args.audio_file}...")
    segments, _ = model.transcribe(args_config["audio_file"], language=args_config["language"])

    # Build content in memory first (much faster)
    content_parts = [
        "WEBVTT\n\n"
        "NOTE\n"
        "This is created by subcut, you can modify as you want, but respect the structure\n"
        "For more ref, use: https://developer.mozilla.org/en-US/docs/Web/API/WebVTT_API/Web_Video_Text_Tracks_Format#cue_payload_text_tags\n\n"
    ]

    # Process segments as they stream in
    for index, segment in enumerate(segments, 1):
        start_time = format_srt_timestamp(segment.start)
        end_time = format_srt_timestamp(segment.end)
        formatted_text = char_parsing_correct_format(segment.text.lstrip())

        # Add segment content
        content_parts.append(f"{index}\n{start_time} --> {end_time}\n{formatted_text}")

        # Add separator (except for last segment)
        if hasattr(segments, '__len__'):  # If we can peek ahead
            content_parts.append("\n\n")
        else:
            # For streaming, always add separator (will trim at end if needed)
            content_parts.append("\n\n")

    # Write all content at once
    with open(args_config["output_file"], "w", buffering=8192) as f:
        f.write("".join(content_parts).rstrip("\n\n"))  # Remove trailing separators

    print("--" * 50)
    return logger.Success("DONE:", [f"file generated at path {args_config['output_file']}"])