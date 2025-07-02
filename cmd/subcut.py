#!/usr/bin/env python3
import argparse
import gen

def main():
    parser = argparse.ArgumentParser(prog="subcut")
    gen.gen_sub_command(parser)

if __name__ == "__main__":
    main()
