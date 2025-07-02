# std modules
import os
from argparse import ArgumentParser
from typing import TextIO
from faster_whisper import WhisperModel

# dev modules
import cuda_check
import defautls

def format_srt_timestamp(seconds:float) -> str:
    h, remainder = divmod(seconds, 3600)
    m, s = divmod(remainder, 60)
    ms = int((s % 1) * 1000)
    return f"{int(h):02d}:{int(m):02d}:{int(s):02d}.{ms:03d}"

def char_parsing_correct_format(text:str) -> str:
    words = text.split(" ")
    new_words = []
    for word in words:
        chars = list(word)
        for i, char in enumerate(chars):
            new_char = char
            match new_char:
                case "&": new_char="&amp;"
                case "<": new_char="&lt;"
                case ">": new_char="&gt;"
                case '"': new_char="&quot;"
                case "'": new_char="&apos;"
                case "–": new_char="&ndash;"
                case "—": new_char="&mdash;"
                case "©": new_char="&copy;"
                case "®": new_char="&reg;"
                case "™": new_char="&trade;"
                case "≈": new_char="&asymp;"
                case "£": new_char="&pound;"
                case "€": new_char="&euro;"
                case "°": new_char="&deg;"
                case _: new_char = char
            chars[i] = new_char
        new_words.append("".join(chars))

    return " ".join(new_words)


def static_content(f: TextIO) -> None:
    content = (
        "WEBVTT\n\n"
        "NOTE\n"
        "This is created by subcut, you can modify as you want, but respect the structure\n"
        "For more ref, use: https://developer.mozilla.org/en-US/docs/Web/API/WebVTT_API/Web_Video_Text_Tracks_Format#cue_payload_text_tags\n\n"
    )
    f.write(content)

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


  # TODO: implement here a runtime issue to be thrown if the args aren't specified in the args_config

  cuda_available = cuda_check.check_cuda_available()

  if cuda_available is None:
        cuda_available = cuda_check.nvidia_msi_check()

  if cuda_available is None:
        cuda_available, error = cuda_check.has_cudart_dll()

  if cuda_available and len(error) == 0:
      args_config["device"] = "cuda"
  else:
      if len(error) > 0 :
          print(f"\033[1;31mERROR:\033[0m {error}")
          os._exit(1)
      args.device = "cpu"
      if defautls.HIGH_END_DEVICES_EN.count(args.model) > 0 or defautls.HIGH_END_DEVICES.count(args.model) > 0:
          print("\033[1;33mWARNING:\033[0m High-end models not recommended. May cause instability on lower-end hardware, use at your own risk")
          choice = input("Use base model for faster results? [y/n]: ").strip().lower()
          if choice == "n":
              args_config["model"] = "base"

  model = WhisperModel(args_config["model"], device=args_config["device"])

  print(f"Transcribing {args.audio_file}...")
  segments, _ = model.transcribe(args_config["audio_file"], language=args_config["language"])

  f = open(f"{args_config["output_file"]}" , "w")
  static_content(f)
  index = 0
  segments_list = list(segments)
  for segment in segments_list:
      index += 1
      start_time = format_srt_timestamp(segment.start)
      end_time = format_srt_timestamp(segment.end)
      formatted_text = char_parsing_correct_format(segment.text)
      block_break = "\n\n"
      if index == len(segments_list) - 1:
          block_break = ""
      f.write(
          f"{index}\n{start_time} --> {end_time}\n{formatted_text.removeprefix(' ')}{block_break}"
      )
  f.close()
  return print(f"\033[1;32mDONE:\033[0m file generated at path {args_config['output_file']}")