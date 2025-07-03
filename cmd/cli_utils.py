
# refactor this to create a logger class to


class Logger:
  MODE_STYLES = {
    "SUCCESS": "\033[1;32m",
    "FAILURE" : "\033[1;31m",
    "WARNING" : "\033[1;33m",
  }

  RESET = "\033[0m"

  def __init__(self) -> None:
    pass

  def Failure(self, highlights: str, word: list[str]) -> None:
    print(f"{self.MODE_STYLES['FAILURE']}{highlights}{self.RESET} {' '.join(word)}")

  def Success(self, highlights: str, word: list[str]) -> None:
      print(f"{self.MODE_STYLES['SUCCESS']}{highlights}{self.RESET} {' '.join(word)}")

  def Warning(self, highlights: str, word: list[str]) -> None:
      print(f"{self.MODE_STYLES['WARNING']}{highlights}{self.RESET} {' '.join(word)}")

