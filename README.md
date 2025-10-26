# blk — A Minimalist Dynamic Systems Language

`blk` is a statically typed, compiled language focused on simplicity, expression-oriented design, and minimal syntax. Inspired by Jai, Zig, Odin, and C. Designed for learning more about compilers.

---

## ✨ Why blk?

- Expression-oriented: every block returns a value
- Minimal syntax: easy to read and parse
- Statically typed, compiler can infer types or you can annotate them
- Compiled: ahead-of-time compilation to efficient machine code
- Simple module system with imports
- Structs, enums, maps, arrays — all built-in
- Powerful block scoping and control flow

---

## 🚀 Example

```blk
import "math"

type User = struct {
    name: string,
    age: u8 = 18,

    greet = fn(user *User) {
        print("Hi, I'm " + user.Name)
    }
}

const main = fn() -> () {
    u := User{ name: "Ali", age: 22 }
    u.greet(u)
    let msg = if u.age > 18 {
        "Adult"
    } else {
        "Minor"
    }
    print(msg)
}
```

## 📄 License

Licensed under the MIT License. See `LICENSE` for more information.
