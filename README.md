# blk — A Minimalist Dynamic Systems Language

`blk` is a dynamically typed, interpreted language focused on simplicity, expression-oriented design, and minimal syntax. Inspired by Jai, Zig, Odin, and C — but reimagined with flexible semantics and runtime evaluation at its core. Designed for quick scripting, tooling, and prototyping with low ceremony and high expressiveness.

---

## ✨ Why blk?

- Expression-oriented: every block returns a value
- Minimal syntax: easy to read and parse
- Dynamically typed, no explicit type declarations
- Interpreted: fast feedback, no build steps required
- Structs, enums, maps, arrays — all built-in
- Powerful block scoping and control flow
- Unified declaration model using `::` and `:=`

---

## 🚀 Example

```blk
import "math"

User :: struct {
    name,
    age,

    greet: fn(self) {
        print("Hi, I'm " + self.name)
    }
}

fn main() {
    u := User{ name: "Ali", age: 22 }
    u.greet()
    msg := if u.age > 18 {
        "Adult"
    } else {
        "Minor"
    }
    print(msg)
}
```

---

## ✅ Language Features

- **Dynamic values**: no static type annotations
- **All variables** declared with `:=`
- **Top-level constants** via `::`
- **Structs with inline methods**
- **Enums**
- **Pattern matching** via `match` expression
- **Expression-based blocks and control flow**
- **Unified literals**: maps and structs share `{}` syntax
- **No distinction between expressions and statements**

---

## 🧱 Core Constructs

### Declarations

```blk
x := 42
msg :: "Welcome to blk"
greet :: fn(name) {
    print("Hello " + name)
}
```

### Structs

```blk
Vec2 :: struct {
    x := 0.0,
    y := 0.0,
    len: fn() {
        sqrt(x * x + y * y)
    }
}

v := Vec2{
    x: 3,
    y: 4
}
print(v.len())
```

### Enums

```blk
Result :: enum {
    Ok,
    Error
}
```

---

## 🔁 Control Flow

### If expressions

```blk
# regular if
name := if loggedIn {
    "User"
} else {
    "Guest"
}

# ternary-like if

user := User{ name: "Alice", age: 22 }

# with use/else tokens
age := if user.age > 18 use "Adult" else "Minor"

# with ?/: tokens
age := if user.age > 18 ? "Adult" : "Minor"

```

### Match expressions

```blk
kind := match x {
    0 => "zero",
    1 => "one",
    _ => "other"
}
```

### While loops

```blk
i := 0
while i < 5 {
    print(i)
    i += 1
}
```

### For loops

```blk
for idx, val in [1, 2, 3] {
    print(idx, val)
}

for k, v in {a: 1, b: 2} {
    print(k, v)
}
```

### next

idea of name `next` suggested by [@gaurangrshah](https://github.com/gaurangrshah)

```blk
while true {
    if shouldSkip() {
        next
    }
    doStuff()
}
```

---

## 📦 Modules & Imports

```blk
import "math"
import "utils"
```

No aliasing needed — always access via `utils.fn`.

---

## 🗃️ Data Types

### Arrays

```blk
nums := [1, 2, 3]
names := ["foo", "bar"]
```

### Maps

```blk
config := {
    "host": "localhost",
    "port": "8080"
}
```

### Struct literals

```blk
person := Person{
    name: "Zed",
    age: 20
}
```

---

## Nul values

this is a special value representing the absence of a value, similar to `null/nil` in other languages. It can be used in any context where a value is expected.

idea of name `nul` suggested by [@unmarine](https://github.com/unmarine)

```blk
x := nul # Represents a null value
```

## 🧠 Expression-Based Semantics

Every code block is an expression. The last expression is the return value of the block — no `return` keyword required.

```blk
double := fn(x) {
    x * 2
}
```

---

## 🧪 Example Evaluation

```blk
result := fn(x, y) {
    if x > y {
        x
    } else {
        y
    }
}

print(result(10, 20))  # 20
```

---

## 📐 Data Shape & Reflection

Types are tracked at runtime via introspection:

```blk
typeOf(x) == "int"
```

---

## 🛠️ Development Roadmap

- [x] Lexer and Tokenizer
- [x] Parser and AST
- [ ] Core Interpreter Engine
- [ ] Error System and Stack Traces
- [ ] REPL and Debugger
- [ ] Built-in Modules (math, io, time, etc.)

---

## ⚙️ Tooling

### Run

```bash
blk run main.blk
```

### Build

```bash
git clone https://github.com/yourname/blk
cd blk
go run cmd/main.go run examples/main.blk
```
