# blk — A Minimalist Systems Programming Language

`blk` is a statically typed, compiled programming language focused on simplicity, predictability, and a clean developer experience. Inspired by languages like Zig, Odin, and C — but designed to be approachable for learners and hobbyists interested in writing small tools, utilities, and experimenting with low-level systems programming.

---

## ✨ Why blk?

- Minimal syntax: easy to read, easy to parse
- Explicit types: no hidden behavior
- No classes or objects — just functions, types, and modules
- Single-pass compilation model
- Stack-based VM (planned) and direct native compilation targets

---

## 🚀 Example

```blk
import "math.blk"

export fn main(): int {
    let result: int = add(5, 7)
    print(result)
    return 0
}

export fn add(a: int, b: int): int {
    return a + b
}
```

---

## ✅ Language Features

- **Static types**: `int`, `float`, `bool`, `string`, `array`, `structs`
- **Functions** with explicit return types and parameters
- **Variables** using `let` (immutable) and `var` (mutable)
- **Custom types** with `type`
- **Import system** using `import "file.blk"`
- **Conditionals**: `if`, `else if`, `else`
- **Loops**: `while`, `for`
- **Explicit memory management**: (planned)
- **No runtime garbage collection**: fully manual or allocator-driven

---

## 📚 Syntax Reference

### Variables

```blk
let x: int = 42
var y: float = 3.14
```

### Functions

```blk
fn double(value: int): int {
    return value * 2
}
```

### Custom Types

```blk
type Point {
    x: float
    y: float
}

fn length(p: Point): float {
    return sqrt(p.x * p.x + p.y * p.y)
}
```

### Conditionals

```blk
if x > 10 {
    print("Greater")
} else {
    print("Smaller or equal")
}
```

### Loops

```blk
var i: int = 0
while i < 10 {
    print(i)
    i = i + 1
}

for n in 0..10 {
    print(n)
}
```

### Imports

```blk
import "math.blk"
import "utils.blk" as utils
```

### Exported Symbols

```blk
export fn add(a: int, b: int): int {
    return a + b
}
```

---

## 🛠️ Development Roadmap

- [x] Lexer and Tokenizer
- [ ] Parser and AST generator
- [ ] Semantic analysis and type checking
- [ ] Bytecode interpreter (stack-based VM)
- [ ] LLVM IR backend
- [ ] Package manager / module system
- [ ] Official standard library (blk stdlib)

---

## ⚙️ Build & Usage

### Compiling blk programs

```bash
blk compile main.blk -o main.blkout
```

### Running blk programs

```bash
blk run main.blk
```

### Building the Compiler (Go example)

```bash
git clone https://github.com/yourname/blk
cd blk
go build -o blk cmd/main.go
```
